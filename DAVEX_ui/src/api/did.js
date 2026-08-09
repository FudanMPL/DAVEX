import request from '@/utils/request'

const actorHeaders = (actorAlias) => {
  const headers = { 'X-Request-ID': `davex-ui-${Date.now()}` }
  if (actorAlias?.trim()) headers['X-DID-Actor'] = actorAlias.trim()
  return headers
}

const get = (url, actorAlias, params = {}) =>
  request.get(url, { params, headers: actorHeaders(actorAlias) })

const post = (url, data, actorAlias) =>
  request.post(url, data, { headers: actorHeaders(actorAlias) })

const put = (url, data, actorAlias) =>
  request.put(url, data, { headers: actorHeaders(actorAlias) })

export const didHealth = () => get('/api/v1/did/health')
export const didReady = (actorAlias) => get('/api/v1/did/ready', actorAlias)
export const didSession = (actorAlias) => get('/api/v1/did/session', actorAlias)
export const didStatus = (actorAlias) => get('/api/v1/did/status', actorAlias)

export const didGenerate = (data, actorAlias) => post('/api/v1/did/identity/generate', data, actorAlias)
export const didRegister = (data, actorAlias) => post('/api/v1/did/identity/register', data, actorAlias)
export const didQuery = (did, actorAlias) => get('/api/v1/did/identity/query', actorAlias, { did })

export const policyRegister = (data, actorAlias) => post('/api/v1/did/policy/register', data, actorAlias)
export const policyQuery = (policyID, actorAlias) => get('/api/v1/did/policy/query', actorAlias, { policyID })
export const policyDeactivate = (id, actorAlias) => post('/api/v1/did/policy/deactivate', { id }, actorAlias)

export const credentialIssue = (data, actorAlias) => post('/api/v1/did/credential/issue', data, actorAlias)
export const credentialQuery = (vcID, actorAlias) => get('/api/v1/did/credential/query', actorAlias, { vcID })
export const credentialVerify = (id, actorAlias) => post('/api/v1/did/credential/verify', { id }, actorAlias)

export const presentationNonce = (data, actorAlias) => post('/api/v1/did/presentation/nonce', data, actorAlias)
export const presentationGenerate = (data, actorAlias) => post('/api/v1/did/presentation/generate', data, actorAlias)
export const presentationVerify = (data, actorAlias) => post('/api/v1/did/presentation/verify', data, actorAlias)

export const roleQuery = (did, actorAlias) => get('/api/v1/did/roles', actorAlias, { did })
export const roleUpdate = (data, actorAlias) => put('/api/v1/did/roles', data, actorAlias)
