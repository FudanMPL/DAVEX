<template>
  <div class="did-workspace">
    <section class="hero-panel">
      <div class="hero-copy">
        <h1>DID 身份与凭证系统</h1>
        <div class="status-row">
          <span class="status-pill" :class="serviceState.http">
            <span class="status-dot"></span>
            Java 接口：{{ statusText(serviceState.http) }}
          </span>
          <span class="status-pill" :class="serviceState.chain">
            <span class="status-dot"></span>
            ChainMaker：{{ statusText(serviceState.chain) }}
          </span>
        </div>
      </div>
      <div class="actor-panel">
        <label>当前链上执行主体</label>
        <el-input
          v-model="actorAlias"
          placeholder="例如 bootstrap / governance"
          clearable
          @change="saveActor"
        />
        <el-button class="default-button refresh-button" :loading="statusLoading" @click="refreshStatus(true)">
          刷新状态
        </el-button>
      </div>
    </section>

    <div class="workspace-grid">
      <section class="operation-panel">
        <el-tabs v-model="activeTab" class="did-tabs">
          <el-tab-pane label="身份 DID" name="identity">
            <div class="tab-intro">
              <h2>身份生命周期</h2>
            </div>

            <el-row :gutter="18">
              <el-col :xs="24" :lg="14">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">生成与注册 DID</span></template>
                  <el-form label-position="top">
                    <el-form-item label="DID">
                      <el-input v-model="identityForm.did" placeholder="did:gov:court-a" />
                    </el-form-item>
                    <el-form-item label="DID Document">
                      <el-input
                        v-model="identityForm.document"
                        type="textarea"
                        :rows="6"
                        placeholder='{"id":"did:gov:court-a","name":"示例主体"}'
                      />
                    </el-form-item>
                    <div class="button-row">
                      <el-button class="default-button" :loading="loading.generateDid" @click="generateIdentity">
                        1. 本地生成
                      </el-button>
                      <el-button class="next-button" :loading="loading.registerDid" @click="registerIdentity">
                        2. 注册上链
                      </el-button>
                    </div>
                  </el-form>
                </el-card>
              </el-col>
              <el-col :xs="24" :lg="10">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">查询身份与角色</span></template>
                  <el-form label-position="top">
                    <el-form-item label="目标 DID">
                      <el-input v-model="identityQueryDid" placeholder="did:gov:court-a" />
                    </el-form-item>
                    <div class="button-stack">
                      <el-button class="default-button" :loading="loading.queryDid" @click="queryIdentity">查询链上 DID</el-button>
                      <el-button class="start-button" :loading="loading.queryRole" @click="queryRoles">查询链上角色</el-button>
                    </div>
                  </el-form>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane label="授权策略" name="policy">
            <div class="tab-intro">
              <h2>授权策略</h2>
            </div>

            <el-row :gutter="18">
              <el-col :xs="24" :lg="14">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">注册策略</span></template>
                  <el-form label-position="top">
                    <el-row :gutter="14">
                      <el-col :span="12"><el-form-item label="颁发者 DID"><el-input v-model="policyForm.issuerDID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="Policy ID"><el-input v-model="policyForm.policyID" /></el-form-item></el-col>
                      <el-col :span="8"><el-form-item label="部门角色"><el-input v-model="policyForm.deptRole" placeholder="court" /></el-form-item></el-col>
                      <el-col :span="8"><el-form-item label="授权范围"><el-input v-model="policyForm.authScope" placeholder="case" /></el-form-item></el-col>
                      <el-col :span="8"><el-form-item label="数据级别"><el-input v-model="policyForm.dataLevel" placeholder="internal" /></el-form-item></el-col>
                      <el-col :span="24"><el-form-item label="动作集合"><el-input v-model="policyForm.actions" placeholder="read,write" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="生效时间（Unix 秒）"><el-input-number v-model="policyForm.validFrom" :min="0" controls-position="right" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="失效时间（Unix 秒）"><el-input-number v-model="policyForm.validUntil" :min="0" controls-position="right" /></el-form-item></el-col>
                    </el-row>
                    <el-button class="default-button" :loading="loading.registerPolicy" @click="registerPolicy">注册策略</el-button>
                  </el-form>
                </el-card>
              </el-col>
              <el-col :xs="24" :lg="10">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">查询与停用</span></template>
                  <el-form label-position="top">
                    <el-form-item label="Policy ID"><el-input v-model="policyQueryId" /></el-form-item>
                    <div class="button-stack">
                      <el-button class="default-button" :loading="loading.queryPolicy" @click="queryPolicy">查询策略</el-button>
                      <el-button type="danger" plain :loading="loading.deactivatePolicy" @click="deactivatePolicy">停用策略</el-button>
                    </div>
                  </el-form>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane label="可验证凭证 VC" name="credential">
            <div class="tab-intro">
              <h2>凭证签发与验证</h2>
            </div>

            <el-row :gutter="18">
              <el-col :xs="24" :lg="14">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">签发 VC</span></template>
                  <el-form label-position="top">
                    <el-row :gutter="14">
                      <el-col :span="12"><el-form-item label="VC ID"><el-input v-model="credentialForm.vcID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="Policy ID"><el-input v-model="credentialForm.policyID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="颁发者 DID"><el-input v-model="credentialForm.issuerDID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="持有者 DID"><el-input v-model="credentialForm.holderDID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="策略条目索引"><el-input-number v-model="credentialForm.entryIndex" :min="0" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="失效时间（Unix 秒）"><el-input-number v-model="credentialForm.expiresAt" :min="0" controls-position="right" /></el-form-item></el-col>
                      <el-col :span="24"><el-form-item><el-checkbox v-model="credentialForm.anonymousEligible">允许用于匿名凭证展示</el-checkbox></el-form-item></el-col>
                    </el-row>
                    <el-button class="default-button" :loading="loading.issueCredential" @click="issueCredential">签发 VC</el-button>
                  </el-form>
                </el-card>
              </el-col>
              <el-col :xs="24" :lg="10">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">查询与验证</span></template>
                  <el-form label-position="top">
                    <el-form-item label="VC ID"><el-input v-model="credentialQueryId" /></el-form-item>
                    <div class="button-stack">
                      <el-button class="default-button" :loading="loading.queryCredential" @click="queryCredential">查询 VC</el-button>
                      <el-button class="next-button" :loading="loading.verifyCredential" @click="verifyCredential">验证 VC</el-button>
                    </div>
                  </el-form>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane label="凭证展示 VP" name="presentation">
            <div class="tab-intro">
              <h2>凭证展示与验证</h2>
            </div>

            <el-row :gutter="18">
              <el-col :xs="24" :lg="12">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">1. 获取 nonce</span></template>
                  <el-form label-position="top">
                    <el-form-item label="验证方 DID"><el-input v-model="nonceForm.verifierDID" /></el-form-item>
                    <el-form-item label="用途"><el-input v-model="nonceForm.purpose" placeholder="case-read" /></el-form-item>
                    <el-form-item label="有效期（秒）"><el-input-number v-model="nonceForm.ttlSeconds" :min="30" :max="3600" /></el-form-item>
                    <el-button class="default-button" :loading="loading.issueNonce" @click="issueNonce">签发 nonce</el-button>
                  </el-form>
                </el-card>
              </el-col>
              <el-col :xs="24" :lg="12">
                <el-card shadow="never" class="operation-card">
                  <template #header><span class="card-title">2. 生成 VP</span></template>
                  <el-form label-position="top">
                    <el-row :gutter="12">
                      <el-col :span="12"><el-form-item label="持有者 DID"><el-input v-model="presentationForm.holderDID" /></el-form-item></el-col>
                      <el-col :span="12"><el-form-item label="验证方 DID"><el-input v-model="presentationForm.verifierDID" /></el-form-item></el-col>
                    </el-row>
                    <el-form-item label="nonce"><el-input v-model="presentationForm.nonce" /></el-form-item>
                    <el-form-item label="用途"><el-input v-model="presentationForm.purpose" /></el-form-item>
                    <el-form-item label="VC IDs"><el-input v-model="presentationForm.vcIDs" placeholder="vc-001,vc-002" /></el-form-item>
                    <el-button class="next-button" :loading="loading.generatePresentation" @click="generatePresentation">生成 VP</el-button>
                  </el-form>
                </el-card>
              </el-col>
              <el-col :span="24">
                <el-card shadow="never" class="operation-card verify-card">
                  <template #header><span class="card-title">3. 验证 VP</span></template>
                  <el-form label-position="top">
                    <el-form-item label="VP JSON">
                      <el-input v-model="verifyPresentationJson" type="textarea" :rows="8" placeholder="粘贴或使用上一步生成的 VP" />
                    </el-form-item>
                    <el-row :gutter="12">
                      <el-col :span="6"><el-form-item label="部门角色"><el-input v-model="accessForm.deptRole" /></el-form-item></el-col>
                      <el-col :span="6"><el-form-item label="授权范围"><el-input v-model="accessForm.authScope" /></el-form-item></el-col>
                      <el-col :span="6"><el-form-item label="数据级别"><el-input v-model="accessForm.dataLevel" /></el-form-item></el-col>
                      <el-col :span="6"><el-form-item label="动作"><el-input v-model="accessForm.action" /></el-form-item></el-col>
                    </el-row>
                    <el-button class="default-button" :loading="loading.verifyPresentation" @click="verifyPresentation">验证 VP</el-button>
                  </el-form>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>
        </el-tabs>
      </section>

      <aside class="result-panel">
        <div class="result-heading">
          <h3>操作结果</h3>
          <el-tag v-if="lastResult" :type="lastResult.code === 0 ? 'success' : 'danger'" effect="dark">
            {{ lastResult.code === 0 ? '成功' : `错误 ${lastResult.code}` }}
          </el-tag>
        </div>
        <div v-if="lastResult" class="result-meta">
          <strong>{{ lastAction }}</strong>
          <code>{{ lastResult.requestId || '—' }}</code>
        </div>
        <pre v-if="lastResult" class="json-viewer">{{ prettyResult }}</pre>
        <div v-else class="empty-result">
          暂无操作结果
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  credentialIssue,
  credentialQuery,
  credentialVerify,
  didGenerate,
  didHealth,
  didQuery,
  didReady,
  didRegister,
  policyDeactivate,
  policyQuery,
  policyRegister,
  presentationGenerate,
  presentationNonce,
  presentationVerify,
  roleQuery,
} from '@/api/did'

const now = Math.floor(Date.now() / 1000)
const activeTab = ref('identity')
const actorAlias = ref(localStorage.getItem('davex-did-actor') || 'bootstrap')
const statusLoading = ref(false)
const serviceState = reactive({ http: 'checking', chain: 'checking' })
const statusMessage = ref('正在检查 DID 服务状态。')
const lastAction = ref('')
const lastResult = ref(null)
const loading = reactive({})

const identityForm = reactive({
  did: 'did:gov:court-a',
  document: '{\n  "id": "did:gov:court-a",\n  "name": "示例法院"\n}',
})
const identityQueryDid = ref('did:gov:court-a')

const policyForm = reactive({
  issuerDID: 'did:gov:court-a',
  policyID: 'policy-case-read',
  deptRole: 'court',
  authScope: 'case',
  dataLevel: 'internal',
  actions: 'read',
  validFrom: now,
  validUntil: now + 7 * 24 * 60 * 60,
})
const policyQueryId = ref('policy-case-read')

const credentialForm = reactive({
  vcID: 'vc-case-read-001',
  holderDID: 'did:gov:holder-a',
  issuerDID: 'did:gov:court-a',
  policyID: 'policy-case-read',
  entryIndex: 0,
  expiresAt: now + 7 * 24 * 60 * 60,
  anonymousEligible: true,
})
const credentialQueryId = ref('vc-case-read-001')

const nonceForm = reactive({ verifierDID: 'did:gov:verifier-a', purpose: 'case-read', ttlSeconds: 120 })
const presentationForm = reactive({
  holderDID: 'did:gov:holder-a',
  verifierDID: 'did:gov:verifier-a',
  nonce: '',
  purpose: 'case-read',
  vcIDs: 'vc-case-read-001',
})
const verifyPresentationJson = ref('')
const accessForm = reactive({ deptRole: 'court', authScope: 'case', dataLevel: 'internal', action: 'read' })

const prettyResult = computed(() => JSON.stringify(lastResult.value, null, 2))

const statusText = (state) => ({ online: '可用', offline: '不可达', disabled: '未启用', checking: '检查中' })[state] || state

const saveActor = () => {
  localStorage.setItem('davex-did-actor', actorAlias.value.trim())
}

const required = (values, message) => {
  if (values.some((value) => value === null || value === undefined || String(value).trim() === '')) {
    ElMessage.warning(message)
    return false
  }
  return true
}

const execute = async (key, title, requestTask) => {
  loading[key] = true
  try {
    const response = await requestTask()
    lastAction.value = title
    lastResult.value = response.data
    if (response.data?.code === 0) {
      ElMessage.success(`${title}成功`)
    } else {
      ElMessage.error(response.data?.message || `${title}失败`)
    }
    return response.data
  } catch (error) {
    const payload = {
      code: -1,
      message: error.response?.data?.message || error.message || '请求失败',
      data: null,
      requestId: null,
    }
    lastAction.value = title
    lastResult.value = payload
    ElMessage.error(`${title}失败：${payload.message}`)
    return payload
  } finally {
    loading[key] = false
  }
}

const refreshStatus = async (notify = false) => {
  statusLoading.value = true
  serviceState.http = 'checking'
  serviceState.chain = 'checking'
  try {
    const health = await didHealth()
    if (health.data?.code === 0) {
      serviceState.http = 'online'
    } else if (health.data?.code === 41001) {
      serviceState.http = 'disabled'
      serviceState.chain = 'disabled'
      statusMessage.value = 'DID 功能未启用'
      return
    } else {
      serviceState.http = 'offline'
    }

    const ready = await didReady(actorAlias.value)
    serviceState.chain = ready.data?.code === 0 ? 'online' : 'offline'
    statusMessage.value = serviceState.chain === 'online' ? 'DID 服务已就绪' : 'ChainMaker 未就绪'
    if (notify) ElMessage[serviceState.chain === 'online' ? 'success' : 'warning'](statusMessage.value)
  } catch (error) {
    serviceState.http = 'offline'
    serviceState.chain = 'offline'
    statusMessage.value = 'DAVEX Java 服务不可达'
    if (notify) ElMessage.warning(statusMessage.value)
  } finally {
    statusLoading.value = false
  }
}

const generateIdentity = async () => {
  if (!required([identityForm.did, identityForm.document], '请填写 DID 和 DID Document')) return
  const result = await execute('generateDid', '生成 DID', () => didGenerate({ ...identityForm }, actorAlias.value))
  if (result?.code === 0) identityQueryDid.value = result.data?.did || identityForm.did
}

const registerIdentity = async () => {
  if (!required([identityForm.did], '请先填写或生成 DID')) return
  await execute('registerDid', '注册 DID', () => didRegister({ did: identityForm.did.trim() }, actorAlias.value))
}

const queryIdentity = async () => {
  if (!required([identityQueryDid.value], '请输入要查询的 DID')) return
  await execute('queryDid', '查询 DID', () => didQuery(identityQueryDid.value.trim(), actorAlias.value))
}

const queryRoles = async () => {
  if (!required([identityQueryDid.value], '请输入要查询的 DID')) return
  await execute('queryRole', '查询角色', () => roleQuery(identityQueryDid.value.trim(), actorAlias.value))
}

const registerPolicy = async () => {
  if (!required([policyForm.issuerDID, policyForm.policyID, policyForm.deptRole, policyForm.authScope, policyForm.dataLevel, policyForm.actions], '请完整填写策略字段')) return
  const actions = policyForm.actions.split(',').map((item) => item.trim()).filter(Boolean)
  const body = {
    issuerDID: policyForm.issuerDID.trim(),
    permissions: [{
      policyID: policyForm.policyID.trim(),
      deptRole: policyForm.deptRole.trim(),
      authScope: policyForm.authScope.trim(),
      dataLevel: policyForm.dataLevel.trim(),
      actionSet: actions,
      validFrom: policyForm.validFrom,
      validUntil: policyForm.validUntil,
    }],
  }
  const result = await execute('registerPolicy', '注册策略', () => policyRegister(body, actorAlias.value))
  if (result?.code === 0) policyQueryId.value = policyForm.policyID
}

const queryPolicy = async () => {
  if (!required([policyQueryId.value], '请输入 Policy ID')) return
  await execute('queryPolicy', '查询策略', () => policyQuery(policyQueryId.value.trim(), actorAlias.value))
}

const deactivatePolicy = async () => {
  if (!required([policyQueryId.value], '请输入 Policy ID')) return
  await execute('deactivatePolicy', '停用策略', () => policyDeactivate(policyQueryId.value.trim(), actorAlias.value))
}

const issueCredential = async () => {
  if (!required([credentialForm.vcID, credentialForm.holderDID, credentialForm.issuerDID, credentialForm.policyID], '请完整填写凭证字段')) return
  const result = await execute('issueCredential', '签发 VC', () => credentialIssue({ ...credentialForm }, actorAlias.value))
  if (result?.code === 0) credentialQueryId.value = credentialForm.vcID
}

const queryCredential = async () => {
  if (!required([credentialQueryId.value], '请输入 VC ID')) return
  await execute('queryCredential', '查询 VC', () => credentialQuery(credentialQueryId.value.trim(), actorAlias.value))
}

const verifyCredential = async () => {
  if (!required([credentialQueryId.value], '请输入 VC ID')) return
  await execute('verifyCredential', '验证 VC', () => credentialVerify(credentialQueryId.value.trim(), actorAlias.value))
}

const issueNonce = async () => {
  if (!required([nonceForm.verifierDID, nonceForm.purpose], '请填写验证方 DID 和用途')) return
  const result = await execute('issueNonce', '签发 nonce', () => presentationNonce({ ...nonceForm }, actorAlias.value))
  if (result?.code === 0) {
    presentationForm.nonce = result.data?.nonce || ''
    presentationForm.verifierDID = nonceForm.verifierDID
    presentationForm.purpose = nonceForm.purpose
  }
}

const generatePresentation = async () => {
  if (!required([presentationForm.holderDID, presentationForm.verifierDID, presentationForm.nonce, presentationForm.purpose, presentationForm.vcIDs], '请完整填写 VP 字段')) return
  const body = {
    holderDID: presentationForm.holderDID.trim(),
    verifierDID: presentationForm.verifierDID.trim(),
    nonce: presentationForm.nonce.trim(),
    purpose: presentationForm.purpose.trim(),
    vcIDs: presentationForm.vcIDs.split(',').map((item) => item.trim()).filter(Boolean),
  }
  const result = await execute('generatePresentation', '生成 VP', () => presentationGenerate(body, actorAlias.value))
  if (result?.code === 0) verifyPresentationJson.value = JSON.stringify(result.data?.vp || result.data, null, 2)
}

const verifyPresentation = async () => {
  if (!required([verifyPresentationJson.value], '请粘贴或生成 VP JSON')) return
  let vp
  try {
    vp = JSON.parse(verifyPresentationJson.value)
  } catch (error) {
    ElMessage.error('VP 不是合法的 JSON')
    return
  }
  const body = {
    vp,
    access: {
      deptRole: accessForm.deptRole.trim(),
      authScope: accessForm.authScope.trim(),
      dataLevel: accessForm.dataLevel.trim(),
      action: accessForm.action.trim(),
      atTime: Math.floor(Date.now() / 1000),
    },
  }
  await execute('verifyPresentation', '验证 VP', () => presentationVerify(body, actorAlias.value))
}

onMounted(() => refreshStatus(false))
</script>

<style scoped lang="scss">
.did-workspace {
  --did-blue: #165ac6;
  --did-border: #dce6f3;
  min-width: 0;
  color: #2d405e;
}

.hero-panel {
  overflow: hidden;
  background: #fff;
  border: 1px solid var(--did-border);
  border-radius: 4px;
}

.hero-copy h1 {
  margin: 0;
  padding: 9px 20px;
  color: #fff;
  font-size: 18px;
  font-weight: 500;
  background: linear-gradient(to right, #005bd8, #65bfff);
}

.status-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding: 16px 20px 8px;
}

.status-pill {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  padding: 7px 12px;
  color: #4f5e7b;
  font-size: 13px;
  background: #f3f6fb;
  border: 1px solid #a9c4df;
  border-radius: 3px;
}

.status-dot {
  width: 8px;
  height: 8px;
  background: #94a3b8;
  border-radius: 50%;
}

.status-pill.online .status-dot { background: #1dc5b3; }
.status-pill.offline .status-dot { background: #f86359; }
.status-pill.disabled .status-dot { background: #f5b923; }

.actor-panel {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 8px 20px 16px;
}

.actor-panel label {
  flex: 0 0 auto;
  color: #4f5e7b;
  font-size: 14px;
  font-weight: 700;
}

.refresh-button {
  flex: 0 0 auto;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 330px;
  gap: 16px;
  align-items: start;
  margin-top: 14px;
}

.operation-panel,
.result-panel {
  background: #fff;
  border: 1px solid var(--did-border);
  border-radius: 4px;
}

.operation-panel { padding: 4px 20px 22px; }

.tab-intro {
  display: flex;
  align-items: center;
  margin: 6px 0 18px;
  padding: 9px 16px;
  color: #fff;
  background: linear-gradient(to right, #005bd8, #65bfff);
}

.tab-intro h2 { margin: 0; color: #fff; font-size: 18px; font-weight: 500; }

.operation-card {
  height: calc(100% - 18px);
  margin-bottom: 18px;
  border-color: #dbe5f1;
}

.card-title { color: #2d405e; font-weight: 700; }
.button-row { display: flex; flex-wrap: wrap; gap: 10px; }
.button-row .el-button { margin-left: 0; }
.button-stack { display: grid; gap: 10px; }
.button-stack .el-button { width: 100%; margin-left: 0; }
.verify-card { height: auto; }

.result-panel {
  position: sticky;
  top: 0;
  min-height: 450px;
  overflow: hidden;
  background: #fff;
  border-color: var(--did-border);
}

.result-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 40px;
  padding: 0 14px 0 20px;
  color: #fff;
  background: linear-gradient(to right, #005bd8, #65bfff);
}

.result-heading h3 { margin: 0; color: #fff; font-size: 18px; font-weight: 500; }
.result-meta { display: grid; gap: 5px; padding: 14px 16px; color: #71809a; font-size: 12px; border-bottom: 1px solid #e4e7ed; }
.result-meta code { overflow: hidden; color: #4f5e7b; text-overflow: ellipsis; }

.json-viewer {
  max-height: 610px;
  margin: 14px;
  padding: 14px;
  overflow: auto;
  color: #4f5e7b;
  font-size: 12px;
  line-height: 1.65;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 3px;
  white-space: pre-wrap;
  word-break: break-word;
}

.empty-result {
  display: grid;
  min-height: 350px;
  padding: 20px;
  place-content: center;
  text-align: center;
}

:deep(.did-tabs > .el-tabs__header) { margin-bottom: 14px; }
:deep(.did-tabs .el-tabs__item) { height: 52px; color: #61718a; font-weight: 700; }
:deep(.did-tabs .el-tabs__item.is-active) { color: #165ac6; }
:deep(.did-tabs .el-tabs__active-bar) { height: 3px; background: #165ac6; }
:deep(.el-card__header) { padding: 14px 16px; background: #fbfdff; }
:deep(.el-card__body) { padding: 17px; }
:deep(.el-form-item__label) { color: #51627c; font-weight: 600; }
:deep(.el-input-number) { width: 100%; }
:deep(.actor-panel .el-input) { border: 0; }

@media (max-width: 1280px) {
  .workspace-grid { grid-template-columns: minmax(0, 1fr); }
  .result-panel { position: static; min-height: 320px; }
  .json-viewer { max-height: 400px; }
}

@media (max-width: 900px) {
  .actor-panel { flex-wrap: wrap; }
}
</style>
