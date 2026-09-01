package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"agentscope/internal/audit"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidRegistration = errors.New("invalid registration")
var ErrInvalidMemberOperation = errors.New("invalid member operation")

type Service struct {
	repo          Repository
	sessionSecret string
	audit         *audit.Service
}

func NewService(repo Repository, sessionSecret string) *Service {
	return &Service{repo: repo, sessionSecret: sessionSecret}
}

func NewServiceWithAudit(repo Repository, sessionSecret string, auditService *audit.Service) *Service {
	return &Service{repo: repo, sessionSecret: sessionSecret, audit: auditService}
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	user, err := s.repo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user == nil || user.Status != UserActive {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	session := &Session{
		ID: sessionID, UserID: user.ID, TenantID: user.TenantID, Role: user.Role,
		ExpiresAt: time.Now().UTC().Add(8 * time.Hour), CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) Register(ctx context.Context, tenantName, email, password string) (*Session, error) {
	tenantName = strings.TrimSpace(tenantName)
	email = strings.ToLower(strings.TrimSpace(email))
	if tenantName == "" || email == "" {
		return nil, ErrInvalidRegistration
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, ErrInvalidRegistration
	}
	repo, ok := s.repo.(interface {
		RegisterTenantOwner(context.Context, string, *User) (string, error)
	})
	if !ok {
		return nil, errors.New("registration repository is not configured")
	}
	user := &User{ID: mustID("usr_"), Email: email, PasswordHash: hash, Role: RoleOwner, Status: UserActive}
	tenantID, err := repo.RegisterTenantOwner(ctx, tenantName, user)
	if err != nil {
		return nil, err
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	session := &Session{ID: sessionID, UserID: user.ID, TenantID: tenantID, Role: user.Role, ExpiresAt: time.Now().UTC().Add(8 * time.Hour), CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ResolveSession(ctx context.Context, sessionID string) (*Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}
	return s.repo.FindSession(ctx, sessionID)
}

func HasPermission(role, permission string) bool {
	if role == RoleOwner {
		return true
	}
	permissions := map[string]map[string]bool{
		RoleAdmin:     {PermissionAgentRead: true, PermissionAgentWrite: true, PermissionRiskRead: true, PermissionMemberRead: true, PermissionMemberWrite: true},
		RoleDeveloper: {PermissionAgentRead: true, PermissionRiskRead: true},
		RoleAuditor:   {PermissionRiskRead: true, PermissionRiskReview: true, PermissionAuditRead: true},
		RoleViewer:    {PermissionAgentRead: true, PermissionRiskRead: true},
	}
	return permissions[role][permission]
}

func (s *Service) ListMembers(ctx context.Context, tenantID string) ([]User, error) {
	return s.repo.ListMembers(ctx, tenantID)
}

type MemberPage struct {
	Items    []User
	Page     int
	PageSize int
	Total    int64
}

func (s *Service) ListMembersPage(ctx context.Context, tenantID string, filter MemberFilter) (MemberPage, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if queryRepo, ok := s.repo.(MemberQueryRepository); ok {
		items, total, err := queryRepo.ListMembersFiltered(ctx, tenantID, filter)
		return MemberPage{Items: items, Page: filter.Offset/filter.Limit + 1, PageSize: filter.Limit, Total: total}, err
	}
	items, err := s.repo.ListMembers(ctx, tenantID)
	return MemberPage{Items: items, Page: 1, PageSize: len(items), Total: int64(len(items))}, err
}

func (s *Service) ChangeMemberRole(ctx context.Context, tenantID, actorID, memberID, role string) error {
	if tenantID == "" || actorID == "" || memberID == "" || actorID == memberID || !validMemberRole(role) {
		return ErrInvalidMemberOperation
	}
	if s.audit != nil {
		record, err := s.audit.Prepare(audit.RecordInput{TenantID: tenantID, ActorID: actorID, Action: "member.role_changed", ResourceType: "user", ResourceID: memberID, After: map[string]any{"role": role}})
		if err != nil {
			return err
		}
		if repo, ok := s.repo.(MemberMutationRepository); ok {
			return repo.UpdateMemberRoleWithAudit(ctx, tenantID, memberID, role, record)
		}
		return errors.New("transactional member mutation repository is not configured")
	}
	return s.repo.UpdateMemberRole(ctx, tenantID, memberID, role)
}

func (s *Service) DisableMember(ctx context.Context, tenantID, actorID, memberID string) error {
	if tenantID == "" || actorID == "" || memberID == "" || actorID == memberID {
		return ErrInvalidMemberOperation
	}
	if s.audit != nil {
		record, err := s.audit.Prepare(audit.RecordInput{TenantID: tenantID, ActorID: actorID, Action: "member.disabled", ResourceType: "user", ResourceID: memberID, After: map[string]any{"status": UserDisabled}})
		if err != nil {
			return err
		}
		if repo, ok := s.repo.(MemberMutationRepository); ok {
			return repo.DisableMemberWithAudit(ctx, tenantID, memberID, record)
		}
		return errors.New("transactional member mutation repository is not configured")
	}
	return s.repo.DisableMember(ctx, tenantID, memberID)
}

func (s *Service) TransferOwner(ctx context.Context, tenantID, actorID, targetID string) error {
	if tenantID == "" || actorID == "" || targetID == "" || actorID == targetID {
		return ErrInvalidMemberOperation
	}
	if s.audit != nil {
		record, err := s.audit.Prepare(audit.RecordInput{TenantID: tenantID, ActorID: actorID, Action: "tenant.owner_transferred", ResourceType: "tenant", ResourceID: tenantID, After: map[string]any{"owner_id": targetID}})
		if err != nil {
			return err
		}
		if repo, ok := s.repo.(MemberMutationRepository); ok {
			return repo.TransferOwnerWithAudit(ctx, tenantID, actorID, targetID, record)
		}
		return errors.New("transactional member mutation repository is not configured")
	}
	return errors.New("owner transfer requires audit-enabled repository")
}

func (s *Service) CreateInvitation(ctx context.Context, tenantID, actorID, email, role string, ttl time.Duration) (*MemberInvitation, string, error) {
	if tenantID == "" || actorID == "" || strings.TrimSpace(email) == "" || !validMemberRole(role) {
		return nil, "", ErrInvalidMemberOperation
	}
	if ttl <= 0 || ttl > 72*time.Hour {
		ttl = 48 * time.Hour
	}
	raw, err := randomToken()
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256([]byte(raw))
	invitation := &MemberInvitation{ID: mustID("inv_"), TenantID: tenantID, Email: strings.ToLower(strings.TrimSpace(email)), Role: role, TokenHash: hex.EncodeToString(hash[:]), InvitedBy: actorID, Status: InvitePending, ExpiresAt: time.Now().UTC().Add(ttl)}
	repo, ok := s.repo.(InvitationRepository)
	if !ok {
		return nil, "", errors.New("invitation repository is not configured")
	}
	if s.audit != nil {
		record, err := s.audit.Prepare(audit.RecordInput{TenantID: tenantID, ActorID: actorID, Action: "member.invitation_created", ResourceType: "invitation", ResourceID: invitation.ID, After: map[string]any{"email": invitation.Email, "role": invitation.Role, "expires_at": invitation.ExpiresAt}})
		if err != nil {
			return nil, "", err
		}
		if mutationRepo, ok := s.repo.(InvitationMutationRepository); ok {
			if err := mutationRepo.CreateInvitationWithAudit(ctx, invitation, record); err != nil {
				return nil, "", err
			}
			return invitation, raw, nil
		}
		return nil, "", errors.New("transactional invitation repository is not configured")
	}
	if err := repo.CreateInvitation(ctx, invitation); err != nil {
		return nil, "", err
	}
	return invitation, raw, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, token, password string) (*Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvitationNotFound
	}
	hash := sha256.Sum256([]byte(token))
	repo, ok := s.repo.(InvitationRepository)
	if !ok {
		return nil, errors.New("invitation repository is not configured")
	}
	invitation, err := repo.FindInvitationByTokenHash(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		return nil, err
	}
	if invitation.Status != InvitePending || !invitation.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvitationUsed
	}
	pwHash, err := HashPassword(password)
	if err != nil {
		return nil, ErrInvalidRegistration
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &User{ID: mustID("usr_"), Email: invitation.Email, PasswordHash: pwHash, Role: invitation.Role, Status: UserActive}
	session := &Session{ID: sessionID, TenantID: invitation.TenantID, UserID: user.ID, Role: invitation.Role, ExpiresAt: now.Add(8 * time.Hour), CreatedAt: now}
	var record *audit.Record
	if s.audit != nil {
		record, err = s.audit.Prepare(audit.RecordInput{TenantID: invitation.TenantID, ActorID: user.ID, Action: "member.invitation_accepted", ResourceType: "invitation", ResourceID: invitation.ID, After: map[string]any{"email": invitation.Email, "role": invitation.Role}})
		if err != nil {
			return nil, err
		}
	}
	if record == nil {
		return nil, errors.New("invitation acceptance requires audit service")
	}
	if err := repo.AcceptInvitation(ctx, invitation, user, session, record); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) LoginOIDC(ctx context.Context, oidcService *OIDCService, identity OIDCIdentity) (*Session, error) {
	if oidcService == nil || identity.Subject == "" || identity.Email == "" {
		return nil, ErrOIDCAccountNotAllowed
	}
	repo, ok := s.repo.(OIDCRepository)
	if !ok {
		return nil, errors.New("oidc repository is not configured")
	}
	user, err := repo.FindByOIDC(ctx, oidcService.Issuer(), identity.Subject)
	if errors.Is(err, ErrUserNotFound) {
		user, err = s.repo.FindByEmail(ctx, identity.Email)
		if err == nil {
			if user.TenantID != oidcService.TenantID() || user.Status != UserActive {
				return nil, ErrOIDCAccountNotAllowed
			}
			if err := repo.BindOIDC(ctx, user.ID, oidcService.Issuer(), identity.Subject); err != nil {
				return nil, err
			}
			issuer, subject := oidcService.Issuer(), identity.Subject
			user.OIDCIssuer, user.OIDCSubject = &issuer, &subject
		} else if errors.Is(err, ErrUserNotFound) {
			role := oidcService.DefaultRole()
			if !validMemberRole(role) {
				role = RoleViewer
			}
			issuer, subject := oidcService.Issuer(), identity.Subject
			user = &User{ID: mustID("usr_"), TenantID: oidcService.TenantID(), Email: identity.Email, Role: role, Status: UserActive, OIDCIssuer: &issuer, OIDCSubject: &subject}
			if err := repo.CreateOIDCUser(ctx, user); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if user.Status != UserActive || user.TenantID != oidcService.TenantID() {
		return nil, ErrOIDCAccountNotAllowed
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	session := &Session{ID: sessionID, UserID: user.ID, TenantID: user.TenantID, Role: user.Role, ExpiresAt: time.Now().UTC().Add(8 * time.Hour), CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ListInvitations(ctx context.Context, tenantID string, offset, limit int) ([]MemberInvitation, int64, error) {
	repo, ok := s.repo.(InvitationRepository)
	if !ok {
		return nil, 0, errors.New("invitation repository is not configured")
	}
	return repo.ListInvitations(ctx, tenantID, offset, limit)
}

func validMemberRole(role string) bool {
	switch role {
	case RoleAdmin, RoleDeveloper, RoleAuditor, RoleViewer:
		return true
	default:
		return false
	}
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomSessionID() (string, error) {
	// Session IDs are stored in VARCHAR(64), including the "ses_" prefix.
	buf := make([]byte, 28)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "ses_" + hex.EncodeToString(buf), nil
}

func mustID(prefix string) string {
	id, err := randomSessionID()
	if err != nil {
		panic(err)
	}
	// Entity IDs are stored in VARCHAR(32); keep the generated identifier
	// comfortably within that limit while retaining enough entropy.
	return prefix + id[4:28]
}

func timeNow() time.Time { return time.Now().UTC() }
