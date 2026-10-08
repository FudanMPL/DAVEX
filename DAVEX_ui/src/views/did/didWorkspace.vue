<template>
  <div class="did-workspace">
    <section class="workspace-panel">
      <div class="actor-switcher">
        <span class="actor-label">当前操作身份</span>
        <el-select
          v-model="actorAlias"
          class="actor-select"
          filterable
          placeholder="请选择操作身份"
          @change="saveActor"
        >
          <el-option
            v-for="actor in actorOptions"
            :key="actor.alias"
            :label="actor.label"
            :value="actor.alias"
          >
            <div class="actor-option">
              <span>{{ actor.name }}</span>
              <small>{{ actor.alias }}</small>
            </div>
          </el-option>
        </el-select>
      </div>

      <el-tabs v-model="activeTab" class="did-tabs">
        <el-tab-pane label="身份管理" name="identity">
          <div class="section-heading">
            <h2>身份管理</h2>
            <el-radio-group v-model="identitySection" size="small">
              <el-radio-button label="did">DID 管理</el-radio-button>
              <el-radio-button label="policy">策略管理</el-radio-button>
            </el-radio-group>
          </div>

          <div
            v-if="identitySection === 'did'"
            class="content-grid content-grid--wide"
          >
            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">创建 DID</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="DID 标识">
                  <el-input
                    v-model="identityForm.did"
                    placeholder="例如 did:gov:court-a"
                  />
                </el-form-item>
                <el-form-item label="身份名称">
                  <el-input
                    v-model="identityName"
                    placeholder="例如 示例法院"
                    @change="syncIdentityDocument"
                  />
                </el-form-item>
                <el-collapse class="advanced-collapse">
                  <el-collapse-item
                    title="高级设置：DID Document"
                    name="document"
                  >
                    <el-input
                      v-model="identityForm.document"
                      type="textarea"
                      :rows="5"
                      placeholder='{"id":"did:gov:court-a","name":"示例法院"}'
                    />
                  </el-collapse-item>
                </el-collapse>
                <div class="button-row card-actions">
                  <el-button
                    class="start-button"
                    :loading="loading.generateDid"
                    @click="generateIdentity"
                  >
                    生成身份材料
                  </el-button>
                  <el-button
                    class="default-button"
                    :loading="loading.registerDid"
                    @click="registerIdentity"
                  >
                    注册 DID
                  </el-button>
                </div>
              </el-form>
            </el-card>

            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">查询 DID</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="目标 DID">
                  <el-input
                    v-model="identityQueryDid"
                    placeholder="输入需要查询的 DID"
                  />
                </el-form-item>
                <div class="button-stack card-actions">
                  <el-button
                    class="default-button"
                    :loading="loading.queryDid"
                    @click="queryIdentity"
                  >
                    查询身份
                  </el-button>
                  <el-button
                    class="start-button"
                    :loading="loading.queryRole"
                    @click="queryRoles"
                  >
                    查询角色
                  </el-button>
                </div>
              </el-form>
            </el-card>
          </div>

          <div v-else class="content-grid content-grid--wide">
            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">登记访问策略</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-row :gutter="14">
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="策略 ID">
                      <el-input v-model="policyForm.policyID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="颁发者 DID">
                      <el-input v-model="policyForm.issuerDID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="8">
                    <el-form-item label="部门角色">
                      <el-input
                        v-model="policyForm.deptRole"
                        placeholder="court"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="8">
                    <el-form-item label="授权范围">
                      <el-input
                        v-model="policyForm.authScope"
                        placeholder="case"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="8">
                    <el-form-item label="数据级别">
                      <el-input
                        v-model="policyForm.dataLevel"
                        placeholder="internal"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="24">
                    <el-form-item label="允许动作">
                      <el-input
                        v-model="policyForm.actions"
                        placeholder="read,write"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>
                <el-collapse class="advanced-collapse">
                  <el-collapse-item title="高级设置：策略有效期" name="period">
                    <el-row :gutter="14">
                      <el-col :xs="24" :sm="12">
                        <el-form-item label="生效时间（Unix 秒）">
                          <el-input-number
                            v-model="policyForm.validFrom"
                            :min="0"
                            controls-position="right"
                          />
                        </el-form-item>
                      </el-col>
                      <el-col :xs="24" :sm="12">
                        <el-form-item label="失效时间（Unix 秒）">
                          <el-input-number
                            v-model="policyForm.validUntil"
                            :min="0"
                            controls-position="right"
                          />
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </el-collapse-item>
                </el-collapse>
                <div class="card-actions">
                  <el-button
                    class="default-button"
                    :loading="loading.registerPolicy"
                    @click="registerPolicy"
                  >
                    登记策略
                  </el-button>
                </div>
              </el-form>
            </el-card>

            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">查询与停用</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="策略 ID">
                  <el-input v-model="policyQueryId" />
                </el-form-item>
                <div class="button-stack card-actions">
                  <el-button
                    class="default-button"
                    :loading="loading.queryPolicy"
                    @click="queryPolicy"
                  >
                    查询策略
                  </el-button>
                  <el-button
                    type="danger"
                    plain
                    :loading="loading.deactivatePolicy"
                    @click="deactivatePolicy"
                  >
                    停用策略
                  </el-button>
                </div>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="可验证凭证管理" name="credential">
          <div class="section-heading">
            <h2>可验证凭证管理</h2>
          </div>

          <div class="content-grid content-grid--wide">
            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">签发可验证凭证</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-row :gutter="14">
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="VC ID">
                      <el-input v-model="credentialForm.vcID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="策略 ID">
                      <el-input v-model="credentialForm.policyID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="颁发者 DID">
                      <el-input v-model="credentialForm.issuerDID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="持有者 DID">
                      <el-input v-model="credentialForm.holderDID" />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="失效时间（Unix 秒）">
                      <el-input-number
                        v-model="credentialForm.expiresAt"
                        :min="0"
                        controls-position="right"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :xs="24" :sm="12">
                    <el-form-item label="策略条目索引">
                      <el-input-number
                        v-model="credentialForm.entryIndex"
                        :min="0"
                        controls-position="right"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="24">
                    <el-form-item class="checkbox-item">
                      <el-checkbox v-model="credentialForm.anonymousEligible">
                        允许该凭证用于匿名认证
                      </el-checkbox>
                    </el-form-item>
                  </el-col>
                </el-row>
                <div class="card-actions">
                  <el-button
                    class="default-button"
                    :loading="loading.issueCredential"
                    @click="issueCredential"
                  >
                    签发 VC
                  </el-button>
                </div>
              </el-form>
            </el-card>

            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">查询与验证凭证</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="VC ID">
                  <el-input
                    v-model="credentialQueryId"
                    placeholder="输入需要查询的 VC ID"
                  />
                </el-form-item>
                <div class="credential-summary">
                  <div>
                    <span>签发状态</span>
                    <strong>等待查询</strong>
                  </div>
                  <div>
                    <span>凭证有效性</span>
                    <strong>等待验证</strong>
                  </div>
                  <div>
                    <span>撤销状态</span>
                    <strong>等待查询</strong>
                  </div>
                </div>
                <div class="button-stack card-actions">
                  <el-button
                    class="start-button"
                    :loading="loading.queryCredential"
                    @click="queryCredential"
                  >
                    查询 VC
                  </el-button>
                  <el-button
                    class="default-button"
                    :loading="loading.verifyCredential"
                    @click="verifyCredential"
                  >
                    验证 VC
                  </el-button>
                </div>
                <el-collapse class="advanced-collapse eligibility-collapse">
                  <el-collapse-item
                    title="匿名资格设置"
                    name="anonymous-membership"
                  >
                    <el-form-item label="匿名资格群组">
                      <el-input
                        v-model="anonymousMembership.groupID"
                        placeholder="输入群组 ID"
                      />
                    </el-form-item>
                    <el-button
                      class="start-button full-button"
                      :loading="loading.addPrivacyMember"
                      @click="addPrivacyMember"
                    >
                      将当前 VC 加入群组
                    </el-button>
                  </el-collapse-item>
                </el-collapse>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="身份与权限联合验证" name="joint">
          <div class="section-heading">
            <h2>身份与权限联合验证</h2>
          </div>

          <el-steps
            :active="jointStep"
            finish-status="success"
            align-center
            class="flow-steps"
          >
            <el-step title="发起联合验证" />
            <el-step title="生成凭证展示" />
            <el-step title="验证身份与权限" />
          </el-steps>

          <div class="content-grid content-grid--three">
            <el-card
              shadow="never"
              class="operation-card flow-card"
              :class="{ 'flow-card--active': jointStep === 0 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">发起联合验证</span>
                  </div>
                  <span class="number-badge">1</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="验证方 DID">
                  <el-input v-model="nonceForm.verifierDID" />
                </el-form-item>
                <el-form-item label="认证用途">
                  <el-input
                    v-model="nonceForm.purpose"
                    placeholder="case-read"
                  />
                </el-form-item>
                <el-form-item label="有效期（秒）">
                  <el-input-number
                    v-model="nonceForm.ttlSeconds"
                    :min="30"
                    :max="600"
                    controls-position="right"
                  />
                </el-form-item>
                <el-button
                  class="default-button full-button"
                  :loading="loading.issueNonce"
                  @click="issueNonce"
                >
                  发起联合验证
                </el-button>
              </el-form>
            </el-card>

            <el-card
              shadow="never"
              class="operation-card flow-card"
              :class="{ 'flow-card--active': jointStep === 1 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">生成凭证展示</span>
                  </div>
                  <span class="number-badge">2</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="持有者 DID">
                  <el-input v-model="presentationForm.holderDID" />
                </el-form-item>
                <el-form-item label="需要出示的 VC">
                  <el-input
                    v-model="presentationForm.vcIDs"
                    placeholder="vc-001,vc-002"
                  />
                </el-form-item>
                <div
                  class="context-chip"
                  :class="{ 'context-chip--ready': presentationForm.nonce }"
                >
                  {{
                    presentationForm.nonce
                      ? '认证信息已自动传入'
                      : '请先发起联合验证'
                  }}
                </div>
                <el-button
                  class="default-button full-button"
                  :loading="loading.generatePresentation"
                  :disabled="!presentationForm.nonce"
                  @click="generatePresentation"
                >
                  生成凭证展示
                </el-button>
              </el-form>
            </el-card>

            <el-card
              shadow="never"
              class="operation-card flow-card"
              :class="{ 'flow-card--active': jointStep === 2 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">提交联合验证</span>
                  </div>
                  <span class="number-badge">3</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item label="部门角色">
                      <el-input v-model="accessForm.deptRole" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="授权范围">
                      <el-input v-model="accessForm.authScope" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="数据级别">
                      <el-input v-model="accessForm.dataLevel" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="访问动作">
                      <el-input v-model="accessForm.action" />
                    </el-form-item>
                  </el-col>
                </el-row>
                <div
                  class="context-chip"
                  :class="{ 'context-chip--ready': verifyPresentationJson }"
                >
                  {{
                    verifyPresentationJson
                      ? 'VP 已自动准备'
                      : '请先生成凭证展示'
                  }}
                </div>
                <el-button
                  class="default-button full-button"
                  :loading="loading.verifyPresentation"
                  :disabled="!verifyPresentationJson"
                  @click="verifyPresentation"
                >
                  验证身份与权限
                </el-button>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="匿名认证" name="anonymous">
          <div class="section-heading">
            <h2>匿名认证</h2>
          </div>

          <div class="anonymous-anchor">
            <span>匿名资格群组</span>
            <el-input
              v-model="privacyPresentationForm.groupID"
              placeholder="输入群组 ID"
            />
          </div>

          <el-steps
            :active="anonymousStep"
            finish-status="success"
            align-center
            class="flow-steps"
          >
            <el-step title="发起匿名认证" />
            <el-step title="生成匿名凭证展示" />
            <el-step title="验证匿名资格" />
          </el-steps>

          <div class="content-grid content-grid--three anonymous-flow">
            <el-card
              shadow="never"
              class="operation-card flow-card"
              :class="{ 'flow-card--active': anonymousStep === 0 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">发起匿名认证</span>
                  </div>
                  <span class="number-badge">1</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="验证方 DID">
                  <el-input v-model="anonymousNonceForm.verifierDID" />
                </el-form-item>
                <el-form-item label="认证用途">
                  <el-input
                    v-model="anonymousNonceForm.purpose"
                    placeholder="case-read"
                  />
                </el-form-item>
                <el-form-item label="有效期（秒）">
                  <el-input-number
                    v-model="anonymousNonceForm.ttlSeconds"
                    :min="30"
                    :max="600"
                    controls-position="right"
                  />
                </el-form-item>
                <el-button
                  class="default-button full-button"
                  :loading="loading.issueAnonymousNonce"
                  @click="issueAnonymousNonce"
                >
                  发起匿名认证
                </el-button>
              </el-form>
            </el-card>

            <el-card
              shadow="never"
              class="operation-card flow-card privacy-vp-card"
              :class="{ 'flow-card--active': anonymousStep === 1 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">生成匿名凭证展示</span>
                  </div>
                  <span class="number-badge">2</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="持有者 DID">
                  <el-input v-model="privacyPresentationForm.holderDID" />
                </el-form-item>
                <el-form-item label="匿名 VP ID">
                  <el-input
                    v-model="privacyPresentationForm.vpID"
                    placeholder="可留空，由系统生成"
                  />
                </el-form-item>
                <div
                  class="context-chip"
                  :class="{
                    'context-chip--ready': privacyPresentationForm.nonce,
                  }"
                >
                  {{
                    privacyPresentationForm.nonce
                      ? '认证信息已自动传入'
                      : '请先发起匿名认证'
                  }}
                </div>
                <el-button
                  class="default-button full-button"
                  :disabled="!privacyPresentationForm.nonce"
                  :loading="loading.generatePrivacyVp"
                  @click="generatePrivacyVp"
                >
                  生成匿名凭证展示
                </el-button>
              </el-form>
            </el-card>

            <el-card
              shadow="never"
              class="operation-card flow-card privacy-verify-card"
              :class="{ 'flow-card--active': anonymousStep === 2 }"
            >
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">验证匿名资格</span>
                  </div>
                  <span class="number-badge">3</span>
                </div>
              </template>
              <el-form label-position="top">
                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item label="部门角色">
                      <el-input v-model="anonymousAccessForm.deptRole" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="授权范围">
                      <el-input v-model="anonymousAccessForm.authScope" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="数据级别">
                      <el-input v-model="anonymousAccessForm.dataLevel" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="访问动作">
                      <el-input v-model="anonymousAccessForm.action" />
                    </el-form-item>
                  </el-col>
                </el-row>
                <div
                  class="context-chip"
                  :class="{ 'context-chip--ready': privacyPresentationJson }"
                >
                  {{
                    privacyPresentationJson
                      ? '匿名凭证展示已自动准备'
                      : '请先生成匿名凭证展示'
                  }}
                </div>
                <el-button
                  class="default-button full-button"
                  :disabled="!privacyPresentationJson"
                  :loading="loading.verifyPrivacyVp"
                  @click="verifyPrivacyVp"
                >
                  验证匿名资格
                </el-button>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="凭证撤销" name="revocation">
          <div class="section-heading">
            <h2>凭证撤销</h2>
          </div>

          <div class="content-grid content-grid--wide">
            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">创建撤销申请</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="待撤销 VC">
                  <el-input
                    v-model="revocationForm.vcID"
                    placeholder="输入 VC ID"
                  />
                </el-form-item>
                <el-form-item label="匿名资格群组">
                  <el-input v-model="revocationForm.groupID" placeholder="输入群组 ID" />
                </el-form-item>
                <el-form-item label="撤销原因">
                  <el-input v-model="revocationForm.eventType" placeholder="输入已登记的事件类型" />
                </el-form-item>
                <el-collapse class="advanced-collapse">
                  <el-collapse-item title="高级设置：事件签发方" name="issuer">
                    <el-form-item label="事件签发方 DID">
                      <el-input v-model="revocationForm.eventIssuerDID" />
                    </el-form-item>
                  </el-collapse-item>
                </el-collapse>
                <el-button
                  class="default-button"
                  :loading="loading.createRevocation"
                  @click="createRevocation"
                >
                  提交撤销申请
                </el-button>
              </el-form>
            </el-card>

            <el-card shadow="never" class="operation-card">
              <template #header>
                <div class="card-header">
                  <div>
                    <span class="card-title">批准与执行</span>
                  </div>
                </div>
              </template>
              <el-form label-position="top">
                <el-form-item label="撤销申请 ID">
                  <el-input
                    v-model="revocationForm.draftID"
                    placeholder="申请创建后自动填入"
                  />
                </el-form-item>
                <div class="approval-box">
                  <div class="approval-title">
                    <span>委员会批准进度</span>
                    <strong>{{ approvalCount }} / {{ approvalThreshold }}</strong>
                  </div>
                  <el-progress
                    :percentage="approvalPercentage"
                    :stroke-width="8"
                    :show-text="false"
                  />
                </div>
                <div class="button-stack card-actions">
                  <el-button
                    class="start-button"
                    :disabled="!revocationForm.draftID || revocationDraft?.status === 'Executed'"
                    :loading="loading.approveRevocation"
                    @click="approveRevocation"
                  >
                    批准申请
                  </el-button>
                  <el-button
                    class="default-button"
                    :disabled="!approvalThreshold || approvalCount < approvalThreshold || revocationDraft?.status === 'Executed'"
                    :loading="loading.executeRevocation"
                    @click="executeRevocation"
                  >
                    执行撤销
                  </el-button>
                  <el-button
                    class="start-button"
                    :loading="loading.queryRevocation"
                    @click="queryRevocation"
                  >
                    查询结果
                  </el-button>
                </div>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </section>

    <section
      v-if="lastResult"
      class="result-card"
      :class="{ 'result-card--error': lastResult.code !== 0 }"
    >
      <div class="result-summary">
        <div class="result-icon">{{ lastResult.code === 0 ? '✓' : '!' }}</div>
        <div>
          <strong>
            {{ lastAction }}{{ lastResult.code === 0 ? '成功' : '失败' }}
          </strong>
          <p>
            {{
              lastResult.message ||
              (lastResult.code === 0 ? '操作已完成' : '请检查请求信息')
            }}
          </p>
        </div>
        <span v-if="lastResult.requestId" class="request-id">
          {{ lastResult.requestId }}
        </span>
      </div>
      <el-collapse class="result-details">
        <el-collapse-item title="查看技术详情" name="json">
          <pre>{{ prettyResult }}</pre>
        </el-collapse-item>
      </el-collapse>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  credentialIssue,
  credentialQuery,
  credentialVerify,
  didGenerate,
  didQuery,
  didRegister,
  didSession,
  policyDeactivate,
  policyQuery,
  policyRegister,
  presentationGenerate,
  presentationNonce,
  presentationVerify,
  privacyGroupMember,
  privacyGroupQuery,
  privacyKeyImage,
  privacyPresentationGenerate,
  privacyPresentationVerify,
  revocationCommitteeQuery,
  revocationConsumed,
  revocationExecute,
  revocationIssuerQuery,
  revocationLogs,
  revocationRequestApprove,
  revocationRequestCreate,
  revocationRequestQuery,
  roleQuery,
} from '@/api/did'

const now = Math.floor(Date.now() / 1000)
const activeTab = ref('identity')
const identitySection = ref('did')
const actorAlias = ref(localStorage.getItem('davex-did-actor') || 'bootstrap')
const actorOptions = ref([
  {
    alias: 'bootstrap',
    name: '系统初始化方',
    label: '系统初始化方（bootstrap）',
  },
  { alias: 'governance', name: '治理方', label: '治理方（governance）' },
])
const lastAction = ref('')
const lastResult = ref(null)
const loading = reactive({})

const identityName = ref('示例法院')
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
const anonymousMembership = reactive({ groupID: 'group-case-read' })

const nonceForm = reactive({
  verifierDID: 'did:gov:verifier-a',
  purpose: 'case-read',
  ttlSeconds: 120,
})
const presentationForm = reactive({
  holderDID: 'did:gov:holder-a',
  verifierDID: 'did:gov:verifier-a',
  nonce: '',
  purpose: 'case-read',
  vcIDs: 'vc-case-read-001',
})
const verifyPresentationJson = ref('')
const accessForm = reactive({
  deptRole: 'court',
  authScope: 'case',
  dataLevel: 'internal',
  action: 'read',
})

const anonymousNonceForm = reactive({
  verifierDID: 'did:gov:verifier-a',
  purpose: 'case-read',
  ttlSeconds: 120,
})
const privacyPresentationForm = reactive({
  vpID: 'pvp-case-read-001',
  groupID: 'group-case-read',
  holderDID: 'did:gov:holder-a',
  verifierDID: 'did:gov:verifier-a',
  nonce: '',
  purpose: 'case-read',
})
const privacyPresentationJson = ref('')
const privacyQualification = ref(null)
const anonymousAccessForm = reactive({
  deptRole: 'court',
  authScope: 'case',
  dataLevel: 'internal',
  action: 'read',
})

const revocationForm = reactive({
  vcID: 'vc-case-read-001',
  groupID: '',
  eventType: '',
  eventIssuerDID: '',
  draftID: '',
})
const revocationDraft = ref(null)
const revocationCommittee = ref(null)
const revocationHash = ref('')
const approvalCount = computed(() => Object.keys(revocationDraft.value?.approvals || {}).length)
const approvalThreshold = computed(() => revocationCommittee.value?.threshold || 0)
const approvalPercentage = computed(() =>
  approvalThreshold.value ? Math.min(100, Math.round(100 * approvalCount.value / approvalThreshold.value)) : 0,
)

const prettyResult = computed(() => JSON.stringify(lastResult.value, null, 2))
const jointStep = computed(() => {
  if (verifyPresentationJson.value) return 2
  if (presentationForm.nonce) return 1
  return 0
})
const anonymousStep = computed(() => {
  if (privacyPresentationJson.value) return 2
  if (privacyPresentationForm.nonce) return 1
  return 0
})

const actorDisplayName = (actor) => {
  const knownNames = {
    bootstrap: '系统初始化方',
    governance: '治理方',
    issuer: '凭证签发方',
    holder: '凭证持有者',
    verifier: '验证方',
  }
  return knownNames[actor.alias] || actor.did || '操作身份'
}

const loadActorOptions = async () => {
  try {
    const response = await didSession(actorAlias.value)
    const actors = response.data?.data?.actors
    if (!Array.isArray(actors) || actors.length === 0) return
    actorOptions.value = actors.map((actor) => {
      const name = actorDisplayName(actor)
      return { ...actor, name, label: `${name}（${actor.alias}）` }
    })
    if (!revocationForm.eventIssuerDID) {
      revocationForm.eventIssuerDID = actors.find((actor) => actor.alias === 'bootstrap')?.did || ''
    }
    if (!actorOptions.value.some((actor) => actor.alias === actorAlias.value)) {
      actorAlias.value = actorOptions.value[0].alias
      saveActor()
    }
  } catch (error) {
    if (!actorOptions.value.some((actor) => actor.alias === actorAlias.value)) {
      actorOptions.value.push({
        alias: actorAlias.value,
        name: '当前身份',
        label: `当前身份（${actorAlias.value}）`,
      })
    }
  }
}

const saveActor = () => {
  localStorage.setItem('davex-did-actor', actorAlias.value)
}

const syncIdentityDocument = () => {
  identityForm.document = JSON.stringify(
    { id: identityForm.did, name: identityName.value },
    null,
    2,
  )
}

const required = (values, message) => {
  if (
    values.some(
      (value) =>
        value === null || value === undefined || String(value).trim() === '',
    )
  ) {
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

const issueAnonymousNonce = async () => {
  if (
    !required(
      [anonymousNonceForm.verifierDID, anonymousNonceForm.purpose],
      '请填写验证方 DID 和认证用途',
    )
  )
    return
  const result = await execute('issueAnonymousNonce', '发起匿名认证', () =>
    presentationNonce({ ...anonymousNonceForm }, actorAlias.value),
  )
  if (result?.code === 0) {
    privacyPresentationJson.value = ''
    privacyQualification.value = null
    privacyPresentationForm.nonce = result.data?.nonce || ''
    privacyPresentationForm.verifierDID = anonymousNonceForm.verifierDID
    privacyPresentationForm.purpose = anonymousNonceForm.purpose
  }
}

const addPrivacyMember = async () => {
  const groupID = anonymousMembership.groupID.trim()
  const vcID = credentialQueryId.value.trim()
  if (!required([groupID, vcID], '请填写群组 ID 和 VC ID')) return
  const group = await execute('queryPrivacyGroup', '查询匿名资格群组', () =>
    privacyGroupQuery(groupID, actorAlias.value),
  )
  if (group?.code !== 0) return
  const result = await execute('addPrivacyMember', '加入匿名资格群组', () =>
    privacyGroupMember({ groupID, vcID }, actorAlias.value),
  )
  if (result?.code === 0) {
    privacyPresentationForm.groupID = groupID
    revocationForm.groupID = groupID
    revocationForm.vcID = vcID
  }
}

const generatePrivacyVp = async () => {
  if (
    !required(
      [
        privacyPresentationForm.groupID,
        privacyPresentationForm.holderDID,
        privacyPresentationForm.verifierDID,
        privacyPresentationForm.nonce,
        privacyPresentationForm.purpose,
      ],
      '请完整填写匿名凭证展示信息',
    )
  )
    return
  const result = await execute('generatePrivacyVp', '生成匿名凭证展示', () =>
    privacyPresentationGenerate({ ...privacyPresentationForm }, actorAlias.value),
  )
  if (result?.code === 0) {
    if (!result.data?.vp || !result.data?.qualification) {
      ElMessage.error('匿名凭证展示响应不完整')
      return
    }
    privacyPresentationJson.value = JSON.stringify(result.data.vp)
    privacyQualification.value = result.data?.qualification || null
  }
}

const verifyPrivacyVp = async () => {
  if (!privacyPresentationJson.value || !privacyQualification.value) {
    ElMessage.warning('请先生成匿名凭证展示')
    return
  }
  const body = {
    vp: JSON.parse(privacyPresentationJson.value),
    qualification: privacyQualification.value,
    access: {
      deptRole: anonymousAccessForm.deptRole.trim(),
      authScope: anonymousAccessForm.authScope.trim(),
      dataLevel: anonymousAccessForm.dataLevel.trim(),
      action: anonymousAccessForm.action.trim(),
      atTime: Math.floor(Date.now() / 1000),
    },
  }
  const result = await execute('verifyPrivacyVp', '验证匿名资格', () =>
    privacyPresentationVerify(body, actorAlias.value),
  )
  if (result?.code === 0 && result.data?.keyImage) {
    const image = await execute('queryPrivacyKeyImage', '查询匿名凭证使用状态', () =>
      privacyKeyImage(result.data.keyImage, actorAlias.value),
    )
    if (image?.code === 0) {
      const verified = result.data.verified === true && image.data?.used === true
      lastAction.value = '匿名资格验证'
      lastResult.value = { code: verified ? 0 : -1,
        message: verified ? 'success' : '匿名验证或密钥映像状态尚未确认',
        requestId: result.requestId,
        data: { verified: result.data.verified, keyImageUsed: image.data?.used,
          keyImage: result.data.keyImage } }
    }
  }
}

watch(
  () => [privacyPresentationForm.groupID, privacyPresentationForm.holderDID,
    privacyPresentationForm.verifierDID, privacyPresentationForm.nonce, privacyPresentationForm.purpose],
  () => {
    privacyPresentationJson.value = ''
    privacyQualification.value = null
  },
)

const generateIdentity = async () => {
  if (
    !required(
      [identityForm.did, identityForm.document],
      '请填写 DID 和 DID Document',
    )
  )
    return
  const result = await execute('generateDid', '生成 DID', () =>
    didGenerate({ ...identityForm }, actorAlias.value),
  )
  if (result?.code === 0)
    identityQueryDid.value = result.data?.did || identityForm.did
}

const registerIdentity = async () => {
  if (!required([identityForm.did], '请先填写或生成 DID')) return
  await execute('registerDid', '注册 DID', () =>
    didRegister({ did: identityForm.did.trim() }, actorAlias.value),
  )
}

const queryIdentity = async () => {
  if (!required([identityQueryDid.value], '请输入要查询的 DID')) return
  await execute('queryDid', '查询 DID', () =>
    didQuery(identityQueryDid.value.trim(), actorAlias.value),
  )
}

const queryRoles = async () => {
  if (!required([identityQueryDid.value], '请输入要查询的 DID')) return
  await execute('queryRole', '查询角色', () =>
    roleQuery(identityQueryDid.value.trim(), actorAlias.value),
  )
}

const registerPolicy = async () => {
  const requiredValues = [
    policyForm.issuerDID,
    policyForm.policyID,
    policyForm.deptRole,
    policyForm.authScope,
    policyForm.dataLevel,
    policyForm.actions,
  ]
  if (!required(requiredValues, '请完整填写策略字段')) return
  const actions = policyForm.actions
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  const body = {
    issuerDID: policyForm.issuerDID.trim(),
    permissions: [
      {
        policyID: policyForm.policyID.trim(),
        deptRole: policyForm.deptRole.trim(),
        authScope: policyForm.authScope.trim(),
        dataLevel: policyForm.dataLevel.trim(),
        actionSet: actions,
        validFrom: policyForm.validFrom,
        validUntil: policyForm.validUntil,
      },
    ],
  }
  const result = await execute('registerPolicy', '登记策略', () =>
    policyRegister(body, actorAlias.value),
  )
  if (result?.code === 0) policyQueryId.value = policyForm.policyID
}

const queryPolicy = async () => {
  if (!required([policyQueryId.value], '请输入策略 ID')) return
  await execute('queryPolicy', '查询策略', () =>
    policyQuery(policyQueryId.value.trim(), actorAlias.value),
  )
}

const deactivatePolicy = async () => {
  if (!required([policyQueryId.value], '请输入策略 ID')) return
  await execute('deactivatePolicy', '停用策略', () =>
    policyDeactivate(policyQueryId.value.trim(), actorAlias.value),
  )
}

const issueCredential = async () => {
  const requiredValues = [
    credentialForm.vcID,
    credentialForm.holderDID,
    credentialForm.issuerDID,
    credentialForm.policyID,
  ]
  if (!required(requiredValues, '请完整填写凭证字段')) return
  const result = await execute('issueCredential', '签发 VC', () =>
    credentialIssue({ ...credentialForm }, actorAlias.value),
  )
  if (result?.code === 0) credentialQueryId.value = credentialForm.vcID
}

const queryCredential = async () => {
  if (!required([credentialQueryId.value], '请输入 VC ID')) return
  await execute('queryCredential', '查询 VC', () =>
    credentialQuery(credentialQueryId.value.trim(), actorAlias.value),
  )
}

const verifyCredential = async () => {
  if (!required([credentialQueryId.value], '请输入 VC ID')) return
  await execute('verifyCredential', '验证 VC', () =>
    credentialVerify(credentialQueryId.value.trim(), actorAlias.value),
  )
}

const issueNonce = async () => {
  if (
    !required(
      [nonceForm.verifierDID, nonceForm.purpose],
      '请填写验证方 DID 和认证用途',
    )
  )
    return
  const result = await execute('issueNonce', '发起联合验证', () =>
    presentationNonce({ ...nonceForm }, actorAlias.value),
  )
  if (result?.code === 0) {
    presentationForm.nonce = result.data?.nonce || ''
    presentationForm.verifierDID = nonceForm.verifierDID
    presentationForm.purpose = nonceForm.purpose
  }
}

const generatePresentation = async () => {
  const values = [
    presentationForm.holderDID,
    presentationForm.verifierDID,
    presentationForm.nonce,
    presentationForm.purpose,
    presentationForm.vcIDs,
  ]
  if (!required(values, '请完整填写凭证展示信息')) return
  const body = {
    holderDID: presentationForm.holderDID.trim(),
    verifierDID: presentationForm.verifierDID.trim(),
    nonce: presentationForm.nonce.trim(),
    purpose: presentationForm.purpose.trim(),
    vcIDs: presentationForm.vcIDs
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
  }
  const result = await execute('generatePresentation', '生成凭证展示', () =>
    presentationGenerate(body, actorAlias.value),
  )
  if (result?.code === 0)
    verifyPresentationJson.value = JSON.stringify(
      result.data?.vp || result.data,
      null,
      2,
    )
}

const verifyPresentation = async () => {
  if (!required([verifyPresentationJson.value], '请先生成凭证展示')) return
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
  await execute('verifyPresentation', '身份与权限联合验证', () =>
    presentationVerify(body, actorAlias.value),
  )
}

const createRevocation = async () => {
  const groupID = revocationForm.groupID.trim()
  const vcID = revocationForm.vcID.trim()
  const eventType = revocationForm.eventType.trim()
  const eventIssuerDID = revocationForm.eventIssuerDID.trim()
  if (!required([groupID, vcID, eventType, eventIssuerDID], '请填写待撤销 VC、群组和事件类型')) return

  const issuer = await execute('queryEventIssuer', '查询撤销事件配置', () =>
    revocationIssuerQuery(eventType, eventIssuerDID, actorAlias.value),
  )
  if (issuer?.code !== 0) return
  if (issuer.data?.status !== 'Active') {
    ElMessage.error('撤销事件签发方尚未激活')
    return
  }
  const committee = await execute('queryRevocationCommittee', '查询撤销委员会', () =>
    revocationCommitteeQuery(groupID, actorAlias.value),
  )
  if (committee?.code !== 0) return
  if (committee.data?.status !== 'Active') {
    ElMessage.error('撤销委员会尚未激活')
    return
  }
  revocationCommittee.value = committee.data

  const result = await execute('createRevocation', '创建撤销申请', () =>
    revocationRequestCreate({ groupID, vcID, eventIssuerDID, eventType,
      scopeType: 'group', scopeID: groupID }, actorAlias.value),
  )
  if (result?.code === 0) {
    revocationDraft.value = result.data
    revocationForm.draftID = result.data?.id || ''
    revocationHash.value = ''
  }
}

const approveRevocation = async () => {
  const draftID = revocationForm.draftID.trim()
  if (!required([draftID], '请先创建或查询撤销申请')) return
  const result = await execute('approveRevocation', '批准撤销申请', () =>
    revocationRequestApprove(draftID, actorAlias.value),
  )
  if (result?.code === 0) revocationDraft.value = result.data?.draft || revocationDraft.value
}

const queryRevocation = async () => {
  const draftID = revocationForm.draftID.trim()
  if (!required([draftID], '请输入撤销申请 ID')) return
  const draftResponse = await execute('queryRevocation', '查询撤销申请', () =>
    revocationRequestQuery(draftID, actorAlias.value),
  )
  if (draftResponse?.code !== 0) return
  const draft = draftResponse.data
  revocationDraft.value = draft
  revocationForm.groupID = draft.groupID
  revocationForm.vcID = draft.vcID
  revocationForm.eventType = draft.credential?.eventType || revocationForm.eventType
  revocationForm.eventIssuerDID = draft.credential?.issuerDID || revocationForm.eventIssuerDID
  const committee = await execute('queryRevocationCommittee', '查询撤销委员会', () =>
    revocationCommitteeQuery(draft.groupID, actorAlias.value),
  )
  if (committee?.code !== 0) return
  revocationCommittee.value = committee.data
  if (draft.status !== 'Executed') {
    lastAction.value = '撤销申请查询'
    lastResult.value = draftResponse
    return
  }
  const vc = await execute('queryRevokedCredential', '查询撤销后 VC', () =>
    credentialQuery(draft.vcID, actorAlias.value),
  )
  if (vc?.code !== 0) return
  const logs = await execute('queryRevocationLogs', '查询撤销记录', () =>
    revocationLogs(draft.vcID, actorAlias.value),
  )
  if (logs?.code !== 0) return
  const hash = revocationHash.value || logs.data?.logs?.[0]?.credentialHash
  let consumed = null
  if (hash) {
    const result = await execute('queryRevocationConsumed', '查询撤销凭证状态', () =>
      revocationConsumed(hash, actorAlias.value),
    )
    if (result?.code !== 0) return
    consumed = result.data
    revocationHash.value = hash
  }
  lastAction.value = '撤销结果查询'
  const complete = vc.data?.proof?.status === 'Revoked'
    && Array.isArray(logs.data?.logs) && logs.data.logs.length > 0
    && consumed?.consumed === true
  lastResult.value = { code: complete ? 0 : -1,
    message: complete ? 'success' : '撤销结果尚未全部确认', requestId: logs.requestId,
    data: { draft, committee: committee.data, vcProof: vc.data?.proof,
      logs: logs.data?.logs, consumed } }
}

const executeRevocation = async () => {
  const draftID = revocationForm.draftID.trim()
  if (!approvalThreshold.value || approvalCount.value < approvalThreshold.value) {
    ElMessage.warning('委员会批准尚未达到门限')
    return
  }
  const result = await execute('executeRevocation', '执行撤销', () =>
    revocationExecute(draftID, actorAlias.value),
  )
  if (result?.code === 0) {
    revocationHash.value = result.data?.credentialHash || ''
    revocationDraft.value = { ...revocationDraft.value, status: 'Executed' }
    await queryRevocation()
  }
}

onMounted(loadActorOptions)
</script>

<style scoped lang="scss">
.did-workspace {
  --did-blue: #165ac6;
  --did-light-blue: #eef6ff;
  --did-border: #dce6f3;
  min-width: 0;
  color: #2d405e;
}

.workspace-panel {
  position: relative;
  padding: 0 20px 22px;
  background: #fff;
  border: 1px solid var(--did-border);
  border-radius: 4px;
}

.actor-switcher {
  position: absolute;
  top: 9px;
  right: 20px;
  z-index: 3;
  display: flex;
  gap: 10px;
  align-items: center;
}

.actor-label {
  flex: 0 0 auto;
  color: #61718a;
  font-size: 13px;
  font-weight: 600;
}

.actor-select {
  width: 220px;
}

.actor-option {
  display: flex;
  justify-content: space-between;
  gap: 18px;
}

.actor-option small {
  color: #97a5b9;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  margin: 4px 0 18px;
}

.section-heading h2 {
  margin: 0;
  color: #2d405e;
  font-size: 20px;
  font-weight: 600;
}

.content-grid {
  display: grid;
  gap: 18px;
  align-items: stretch;
}

.content-grid--wide {
  grid-template-columns: minmax(0, 1.45fr) minmax(300px, 0.75fr);
}

.content-grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.operation-card {
  height: 100%;
  border-color: #dbe5f1;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.card-header > div {
  display: grid;
  gap: 4px;
}

.card-title {
  color: #2d405e;
  font-size: 15px;
  font-weight: 700;
}

.number-badge {
  flex: 0 0 auto;
  min-width: 36px;
  padding: 4px 9px;
  color: #165ac6;
  font-size: 12px;
  text-align: center;
  background: #eaf3ff;
  border-radius: 2px;
}

.number-badge {
  min-width: 24px;
  width: 24px;
  height: 24px;
  padding: 0;
  color: #fff;
  line-height: 24px;
  background: #7ea9e7;
  border-radius: 50%;
}

.button-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.button-row .el-button,
.button-stack .el-button {
  margin-left: 0;
}

.button-stack {
  display: grid;
  gap: 10px;
}

.button-stack .el-button,
.full-button {
  width: 100%;
}

.card-actions {
  margin-top: 20px;
}

.credential-summary {
  display: grid;
  gap: 1px;
  overflow: hidden;
  background: #e7edf5;
  border: 1px solid #e7edf5;
  border-radius: 3px;
}

.credential-summary div {
  display: flex;
  justify-content: space-between;
  padding: 11px 13px;
  color: #74849b;
  font-size: 12px;
  background: #fbfcfe;
}

.credential-summary strong {
  color: #4e607b;
  font-weight: 500;
}

.advanced-collapse {
  margin-top: 2px;
  border-bottom: 0;
}

.eligibility-collapse {
  margin-top: 14px;
}

.flow-steps {
  margin: 8px 3% 26px;
}

.flow-card {
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.flow-card--active {
  border-color: #80b6f4;
  box-shadow: 0 4px 14px rgba(45, 105, 190, 0.08);
}

.flow-card--active .number-badge {
  background: #165ac6;
}

.context-chip {
  margin: 2px 0 14px;
  padding: 9px 12px;
  color: #9a7b36;
  font-size: 12px;
  text-align: center;
  background: #fff9e9;
  border: 1px solid #f4dfad;
  border-radius: 3px;
}

.context-chip--ready {
  color: #168176;
  background: #eaf9f7;
  border-color: #afe2db;
}

.anonymous-anchor {
  display: flex;
  gap: 12px;
  align-items: center;
  max-width: 620px;
  margin: 0 auto 24px;
  padding: 12px 16px;
  background: #f6f9fd;
  border: 1px solid #dbe5f1;
  border-radius: 4px;
}

.anonymous-anchor > span:first-child {
  flex: 0 0 auto;
  color: #51627c;
  font-size: 13px;
  font-weight: 600;
}

.approval-box {
  padding: 16px;
  background: #f7faff;
  border: 1px solid #e0e9f4;
  border-radius: 4px;
}

.approval-title {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
  color: #5a6e8b;
  font-size: 13px;
}

.approval-title strong {
  color: #165ac6;
}

.full-select {
  width: 100%;
}

.result-card {
  margin-top: 14px;
  overflow: hidden;
  background: #fff;
  border: 1px solid #b8e2dc;
  border-radius: 4px;
}

.result-card--error {
  border-color: #f0c1bd;
}

.result-summary {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 15px 18px;
  background: #f0faf8;
}

.result-card--error .result-summary {
  background: #fff5f4;
}

.result-icon {
  flex: 0 0 auto;
  width: 30px;
  height: 30px;
  color: #fff;
  font-weight: 700;
  line-height: 30px;
  text-align: center;
  background: #1db4a4;
  border-radius: 50%;
}

.result-card--error .result-icon {
  background: #e66b62;
}

.result-summary > div:nth-child(2) {
  min-width: 0;
}

.result-summary strong {
  color: #354a69;
  font-size: 14px;
}

.result-summary p {
  margin: 3px 0 0;
  color: #7a899f;
  font-size: 12px;
}

.request-id {
  margin-left: auto;
  overflow: hidden;
  color: #8b99ab;
  font-family: monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-details {
  padding: 0 18px;
  border-top: 1px solid #e5ecf4;
  border-bottom: 0;
}

.result-details pre {
  max-height: 360px;
  margin: 0;
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

:deep(.did-tabs > .el-tabs__header) {
  margin: 0 285px 18px 0;
}

:deep(.did-tabs .el-tabs__item) {
  height: 52px;
  padding: 0 18px;
  color: #61718a;
  font-size: 14px;
  font-weight: 600;
}

:deep(.did-tabs .el-tabs__item.is-active) {
  color: #165ac6;
}

:deep(.did-tabs .el-tabs__active-bar) {
  height: 3px;
  background: #165ac6;
}

:deep(.el-card__header) {
  padding: 14px 16px;
  background: #fbfdff;
}

:deep(.el-card__body) {
  padding: 18px;
}

:deep(.el-form-item__label) {
  color: #51627c;
  font-weight: 600;
}

:deep(.el-input-number) {
  width: 100%;
}

:deep(.default-button.is-disabled),
:deep(.default-button.is-disabled:hover) {
  color: #9aa8ba !important;
  background: #edf1f6 !important;
  box-shadow: none !important;
  cursor: not-allowed;
}

:deep(.advanced-collapse .el-collapse-item__header) {
  height: 42px;
  color: #6f7f96;
  font-size: 12px;
}

:deep(.advanced-collapse .el-collapse-item__wrap) {
  border-bottom: 0;
}

:deep(.advanced-collapse .el-collapse-item__content) {
  padding: 4px 0 2px;
}

:deep(.checkbox-item) {
  margin-bottom: 0;
}

:deep(.flow-steps .el-step__title) {
  color: #455b79;
  font-size: 14px;
}

:deep(.flow-steps .el-step__description) {
  color: #8b99ac;
  font-size: 12px;
}

@media (max-width: 1480px) {
  :deep(.did-tabs .el-tabs__item) {
    padding: 0 10px;
    font-size: 13px;
  }

  .content-grid--three {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1160px) {
  .actor-switcher {
    position: static;
    justify-content: flex-end;
    padding: 12px 0 0;
  }

  :deep(.did-tabs > .el-tabs__header) {
    margin-right: 0;
  }

  .content-grid--wide {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .workspace-panel {
    padding: 0 12px 16px;
  }

  .actor-switcher,
  .section-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .anonymous-anchor {
    align-items: stretch;
    flex-direction: column;
    max-width: none;
  }

  .actor-select {
    width: 100%;
  }

  :deep(.did-tabs .el-tabs__nav-wrap) {
    overflow-x: auto;
  }

  .request-id {
    display: none;
  }
}
</style>
