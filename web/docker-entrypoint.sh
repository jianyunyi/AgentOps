#!/bin/sh
set -eu

node -e 'const fs=require("fs"); const value=process.env.NEXT_PUBLIC_API_BASE_URL || ""; fs.mkdirSync("/app/public", {recursive:true}); fs.writeFileSync("/app/public/runtime-config.js", `window.__AGENTOPS_RUNTIME_CONFIG__ = ${JSON.stringify({apiBaseUrl:value})};\n`)'
exec node server.js
