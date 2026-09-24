<template>
  <el-container>
    <span style="display: block; margin-bottom: 10px;">代理</span>
    <el-select v-model="agentId" placeholder="选择代理" @change="handleSelectAgent">
      <el-option
          v-for="item in agents"
          :key="item.value"
          :label="item.label"
          :value="item.value"
      />
    </el-select>
    <el-header class="custom-header">
<!--      <div-->
<!--          style="-->
<!--          background-color: antiquewhite;-->
<!--          height: 40px;-->
<!--          display: flex;-->
<!--          justify-content: center;-->
<!--          align-items: center;-->
<!--        "-->
<!--      >-->
<!--        <p-->
<!--            style="-->
<!--            font-size: 20px;-->
<!--            color: black;-->
<!--            opacity: 100%;-->
<!--            text-align: center;-->
<!--          "-->
<!--        >-->
<!--          文件列表-->
<!--        </p>-->
<!--      </div>-->
      <div class="icon-text">
        <el-icon><Folder /></el-icon>
        <span>文件列表</span>
      </div>
    </el-header>
    <el-main>
      <div>
        <el-button class="default-button" @click="getDirectoryMethod">返回根目录</el-button>
        <el-button class="default-button" @click="returnFrontDirectory">返回上一级目录</el-button>
      </div>
      <div>
        <el-table
            :data="directoryData"
            @row-dblclick="handleCellDoubleClick"
            max-height="400"
        >
          <el-table-column fixed label="" width="50" align="center">
            <template #default="scope">
              <!-- 根据 scope.row.type 的值来决定显示哪个图标 -->
              <el-icon>
                <template v-if="scope.row.type === 'folder'">
                  <el-icon color="#409efc"><Folder /></el-icon>
                </template>
                <template v-else-if="scope.row.type === 'file'">
                  <el-icon><Files /></el-icon>
                </template>
              </el-icon>
            </template>
          </el-table-column>
          <el-table-column
              label="文件ID"
              prop="uid"
              width="200"
              align="center"
          ></el-table-column>
          <el-table-column
              label="名称"
              prop="name"
              width="180"
              align="center"
          ></el-table-column>
          <el-table-column
              label="所属代理"
              width="200"
              align="center"
          >
            <template #default="scope">
              {{ getAgentName(scope.row.agentId) }}
            </template>
          </el-table-column>

          <el-table-column
              label="类型"
              prop="type"
              width="180"
              align="center"
          ></el-table-column>
          <el-table-column
              label="创建时间"
              prop="createDate"
              width="300"
              :formatter="formatDate"
              align="center"
          ></el-table-column>
          <el-table-column
              label="更新时间"
              prop="lastUpdate"
              width="300"
              :formatter="formatDate"
              align="center"
          ></el-table-column>
          <el-table-column
              fixed="right"
              label="操作"
              min-width="200"
              header-align="center"
              align="center"
          >
            <template v-slot="scope">
              <el-button
                  class="small-default-button"
                  v-if="scope.row.type === 'file'"
                  :loading="transferringFileId === scope.row.uid"
                  :disabled="Boolean(transferringFileId)"
                  @click="getFileMethod(scope.row)"
              >
                <el-icon><Download /></el-icon> 传输文件
              </el-button>
<!--              <el-button-->
<!--                  v-if="scope.row.type === 'file' && !isAccessible(scope.row.ruleList)"-->
<!--                  link-->
<!--                  type="danger"-->
<!--                  size="small"-->
<!--                  disabled-->
<!--              >-->
<!--                无权获取文件-->
<!--              </el-button>-->
            </template>
          </el-table-column>
        </el-table>
      </div>

      <el-dialog v-model="transSuccessVisible" title="文件传输结果" width="30%">
        <div>文件传输完成，文件已保存到结果管理区。</div>
        <div v-if="downloadErrorMessage" class="download-error">
          {{ downloadErrorMessage }}
        </div>
        <template #footer>
          <div class="dialog-footer">
            <el-button
                class="default-button"
                :loading="previewLoading"
                :disabled="downloadLoading"
                @click="previewTransferredFile"
            >
              <el-icon><View /></el-icon> 文件预览
            </el-button>
            <el-button
                class="default-button"
                :loading="downloadLoading"
                :disabled="previewLoading"
                @click="downloadTransferredFile"
            >
              立即下载
            </el-button>
          </div>
        </template>
      </el-dialog>
      <el-dialog
          v-model="previewVisible"
          title="文件预览"
          width="70%"
          append-to-body
      >
        <pre class="file-preview-content">{{ previewContent }}</pre>
        <template #footer>
          <div class="dialog-footer">
            <el-button class="close-button" @click="previewVisible = false">关闭</el-button>
          </div>
        </template>
      </el-dialog>
      <el-dialog v-model="transFailedVisible" title="文件传输结果" width="30%">
        <span>{{ transFailedMessage }}</span>
        <template #footer>
          <div class="dialog-footer">
            <el-button class="close-button" @click="transFailedVisible = false">返回</el-button>
          </div>
        </template>
      </el-dialog>
    </el-main>
  </el-container>
</template>

<script lang="ts" setup>
import {getAgent} from '../../api/testDve.js'
import {getDirectory, getRootByAgent} from '../../api/folderController.js'
import {fetchFileByHttp, getFile, getResult, readFile} from "../../api/direct.js";
import {onMounted, ref} from "vue";
import {Download, View} from "@element-plus/icons-vue";
import {ElLoading, ElMessage} from "element-plus";

onMounted(() => {
  getAgentMethod()
  // getDirectoryMethod()
})

// 对话框是否可见
const transSuccessVisible = ref(false)
const transFailedVisible = ref(false)
const transFailedMessage = ref('');
const transferringFileId = ref('')
const downloadLoading = ref(false)
const previewLoading = ref(false)
const previewVisible = ref(false)
const previewContent = ref('')
const downloadErrorMessage = ref('')
const transferredFile = ref(null)
const transferredOutputId = ref(null)

const agents = ref([])
const agentId = ref('')
const applicationId = "DAVEX-C1-A1"
const directoryData = ref([])
const currentDirectoryData = ref([])
// const getDirectoryBody = ref({
//   applicationId: applicationId,
//   agentId: '5'
// })
const getDirectoryBody = ref({
  rootId: '',
})
const folderRoute = ref([])
const getFileBody = ref({
  fileId: '',
  agentId: '',
  folderId: '',
  applicationId: applicationId
})

function addFolderRoute(row) {
  folderRoute.value.push(row.uid, row.name)
}
function deleteFolderRoute() {
  folderRoute.value.pop()
  folderRoute.value.pop()
}

const getDirectoryMethod = async () => {
  try {
    const res = await getDirectory(getDirectoryBody.value)
    const res1 = res.data.data.children
    directoryData.value = res1
    currentDirectoryData.value = '1'
    addFolderRoute(res.data.data)
  } catch (error) {
    console.error('Failed to get group list:', error)
  }
}

const locateTransferredOutput = async (fileInfo) => {
  const res = await getResult({applicationId})
  const outputs = Array.isArray(res.data?.data) ? res.data.data : []
  const matches = outputs
      .filter(item => item.fileId === fileInfo.uid && item.agentId === fileInfo.agentId)
      .sort((first, second) => Number(second.uploadDate || 0) - Number(first.uploadDate || 0))
  return matches[0]?.uid ?? null
}

const getErrorMessage = (error, fallback) => {
  return error?.response?.data?.message || error?.message || fallback
}

const getFileMethod = async (fileInfo) => {
  if (transferringFileId.value) return

  const loadingInstance = ElLoading.service({
    lock: true,
    text: '文件传输中，请稍候……',
    background: 'rgba(0, 0, 0, 0.35)',
  })

  transferringFileId.value = fileInfo.uid
  transferredFile.value = fileInfo
  transferredOutputId.value = null
  previewVisible.value = false
  previewContent.value = ''
  downloadErrorMessage.value = ''
  transFailedVisible.value = false

  try {
    getFileBody.value.fileId = fileInfo.uid
    getFileBody.value.agentId = fileInfo.agentId
    getFileBody.value.folderId = fileInfo.parentId
    const res = await getFile(getFileBody.value)
    if (res.data.code == 1) {
      try {
        transferredOutputId.value = await locateTransferredOutput(fileInfo)
        if (transferredOutputId.value === null) {
          downloadErrorMessage.value = '暂未定位到传输结果，请稍后在结果管理区下载。'
        }
      }
      catch (error) {
        console.error('Failed to locate transferred output:', error)
        downloadErrorMessage.value = '暂未定位到传输结果，请稍后在结果管理区下载。'
      }
      transSuccessVisible.value = true
    }
    else {
      transFailedMessage.value = res.data.message || '文件传输失败';
      transFailedVisible.value = true
    }
  }
  catch (error) {
    console.error('Failed to get file:', error)
    transFailedMessage.value = getErrorMessage(error, '文件传输失败，请稍后重试。')
    transFailedVisible.value = true
  }
  finally {
    transferringFileId.value = ''
    loadingInstance.close()
  }
}

const getDownloadFileName = (contentDisposition, fallbackName) => {
  if (!contentDisposition) return fallbackName

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  const normalMatch = contentDisposition.match(/filename="?([^";]+)"?/i)
  const encodedName = utf8Match?.[1] || normalMatch?.[1]
  if (!encodedName) return fallbackName

  try {
    return decodeURIComponent(encodedName)
  }
  catch (error) {
    return encodedName
  }
}

const decodeTextFile = async (fileBlob) => {
  const fileBuffer = await fileBlob.arrayBuffer()
  try {
    return new TextDecoder('utf-8', {fatal: true}).decode(fileBuffer).replace(/^\uFEFF/, '')
  }
  catch (error) {
    return new TextDecoder('gb18030').decode(fileBuffer).replace(/^\uFEFF/, '')
  }
}

const previewTextFile = async () => {
  const res = await fetchFileByHttp({
    outputId: transferredOutputId.value,
    applicationId,
  })
  const contentType = res.headers?.['content-type'] || ''
  if (contentType.includes('application/json')) {
    const responseText = await res.data.text()
    let message = '文件预览失败，请稍后重试。'
    try {
      message = JSON.parse(responseText)?.message || message
    }
    catch (error) {
      if (responseText) message = responseText
    }
    throw new Error(message)
  }

  return decodeTextFile(res.data)
}

const previewTransferredFile = async () => {
  if (previewLoading.value) return

  if (transferredOutputId.value === null) {
    downloadErrorMessage.value = '暂未定位到传输结果，请稍后在结果管理区预览。'
    ElMessage.warning(downloadErrorMessage.value)
    return
  }

  previewLoading.value = true
  downloadErrorMessage.value = ''
  try {
    const fileName = transferredFile.value?.name || ''
    const extension = fileName.includes('.') ? fileName.split('.').pop().toLowerCase() : ''
    if (['txt', 'text', 'csv'].includes(extension)) {
      previewContent.value = await previewTextFile()
    }
    else {
      const res = await readFile({
        outputId: transferredOutputId.value,
        applicationId,
      })
      if (res.data?.code !== 1) {
        throw new Error(res.data?.message || '文件预览失败，请稍后重试。')
      }
      previewContent.value = String(res.data?.data ?? '')
    }

    previewVisible.value = true
  }
  catch (error) {
    console.error('Failed to preview file:', error)
    downloadErrorMessage.value = getErrorMessage(error, '文件预览失败，请稍后重试。')
    ElMessage.error(downloadErrorMessage.value)
  }
  finally {
    previewLoading.value = false
  }
}

const downloadTransferredFile = async () => {
  if (downloadLoading.value) return

  if (transferredOutputId.value === null) {
    downloadErrorMessage.value = '暂未定位到传输结果，请稍后在结果管理区下载。'
    ElMessage.warning(downloadErrorMessage.value)
    return
  }

  downloadLoading.value = true
  downloadErrorMessage.value = ''
  try {
    const res = await fetchFileByHttp({
      outputId: transferredOutputId.value,
      applicationId,
    })
    const contentType = res.headers?.['content-type'] || ''
    if (contentType.includes('application/json')) {
      const responseText = await res.data.text()
      let message = '下载失败，请稍后重试。'
      try {
        message = JSON.parse(responseText)?.message || message
      }
      catch (error) {
        if (responseText) message = responseText
      }
      throw new Error(message)
    }

    const fileName = getDownloadFileName(
        res.headers?.['content-disposition'],
        transferredFile.value?.name || 'download'
    )
    const objectUrl = URL.createObjectURL(res.data)
    const link = document.createElement('a')
    link.href = objectUrl
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(objectUrl)

    transSuccessVisible.value = false
    ElMessage.success('文件下载已开始')
  }
  catch (error) {
    console.error('Failed to download file:', error)
    downloadErrorMessage.value = getErrorMessage(error, '下载失败，请稍后重试。')
    ElMessage.error(downloadErrorMessage.value)
  }
  finally {
    downloadLoading.value = false
  }
}

const findCurrentFolder = async () => {
  //沿着folderRoute找到当前文件夹
  const res1 = await getDirectory(getDirectoryBody.value)
  const res = res1.data.data.children
  currentDirectoryData.value = res
  if (folderRoute.value.length === 1) {
    directoryData.value = res
    return
  }
  for (let i = 1; i < folderRoute.value.length; i++) {
    for (let j = 0; j < currentDirectoryData.value.length; j++) {
      if (currentDirectoryData.value[j].uid === folderRoute.value[i]) {
        currentDirectoryData.value = currentDirectoryData.value[j].children
        break
      }
    }
  }
  directoryData.value = currentDirectoryData.value
}

const returnFrontDirectory = async () => {
  deleteFolderRoute()
  console.log('folderRoute:', folderRoute.value)
  findCurrentFolder()
}

const currentParentId = ref('1')
const handleCellDoubleClick = async (row) => {
  if (row.type === 'folder') {
    addFolderRoute(row)
    console.log('folderRoute:', folderRoute.value)
    currentParentId.value = row.uid
    directoryData.value = row.children
  }
}

const formatDate = (row, column, cellValue) => {
  if (!cellValue) return ''
  const date = new Date(cellValue)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

const isAccessible = (ruleList) => {
  return ruleList.includes('direct')
}

const getRootByAgentMethod = async (agentId) => {
  const res = await getRootByAgent(agentId)
  getDirectoryBody.value.rootId = res.data.data.uid
}

const getAgentMethod = async () => {
  const res = await getAgent()
  agents.value = res.data.body.data.map(item => ({
    value: item.uid,
    label: item.name || item.uid
  }))
}

const getAgentName = (currentAgentId) => {
  return agents.value.find(item => item.value === currentAgentId)?.label || currentAgentId
}

const handleSelectAgent = async (value) => {
  agentId.value = value
  await getRootByAgentMethod(agentId.value)
  getDirectoryMethod()
}
</script>

<style scoped>
.download-error {
  margin-top: 12px;
  color: var(--el-color-danger);
}

.file-preview-content {
  box-sizing: border-box;
  max-height: 60vh;
  margin: 0;
  overflow: auto;
  padding: 16px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-family: inherit;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
