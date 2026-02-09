# API

All endpoints are local-only and require the API token.

**Auth**
Send the token in `Authorization` header or `?token=` query parameter.

**Endpoints**
1. `GET /health`  
   Returns `{"status":"ok"}`.
2. `GET /status`  
   Returns `{"status":"running"}`.
3. `GET /scanners`  
   Returns available plugins and scheduled jobs.
4. `POST /scanners/trigger/{plugin}`  
   Runs a plugin once and returns the result.
5. `GET /results/latest`  
   Returns latest result per plugin.
6. `GET /results/history`  
   Returns recent history.
7. `GET /findings`  
   Returns recent findings.
8. `GET /baselines`  
   Returns current baselines.

The same endpoints are available under `/api/*`.
