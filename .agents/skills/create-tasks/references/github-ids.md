# Cached GitHub node IDs (LoteoApp/LoteosAPP)

These rarely change, so they're cached here to skip a GraphQL round trip on
every run. If a mutation using one of these IDs fails with a "could not
resolve to a node" or permissions error, re-fetch it with the query below
before assuming something else is wrong — the project or labels may have
been recreated.

| What | ID |
|---|---|
| Org Project v2 "LoteosAPP" (org `LoteoApp`, project number 2) | `PVT_kwDOEoypDc4BfWHQ` |
| Label `epic` | `LA_kwDOTs9m0M8AAAACu1E2pw` |
| Label `task` | `LA_kwDOTs9m0M8AAAACu1E2yg` |

Re-fetch query if needed:

```bash
curl -s -X POST https://api.github.com/graphql \
  -H "Authorization: bearer $TOKEN" \
  -d '{"query":"query { organization(login: \"LoteoApp\") { projectV2(number: 2) { id title } } repository(owner: \"LoteoApp\", name: \"LoteosAPP\") { label(name: \"epic\") { id name } taskLabel: label(name: \"task\") { id name } } }"}'
```

If the numbers drift (new project number, renamed labels), update this file
in the same PR that discovers the change — that's the whole point of caching
it here instead of in a one-off shell history.
