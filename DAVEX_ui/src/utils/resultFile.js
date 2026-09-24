export const getDownloadFileName = (contentDisposition, fallbackName) => {
  if (!contentDisposition) return fallbackName

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  const normalMatch = contentDisposition.match(/filename="?([^";]+)"?/i)
  const encodedName = utf8Match?.[1] || normalMatch?.[1]
  if (!encodedName) return fallbackName

  try {
    return decodeURIComponent(encodedName)
  } catch (error) {
    return encodedName
  }
}

export const assertFileResponse = async (response, fallbackMessage) => {
  const contentType = response.headers?.['content-type'] || ''
  if (!contentType.includes('application/json')) return

  const responseText = await response.data.text()
  let message = fallbackMessage
  try {
    message = JSON.parse(responseText)?.message || message
  } catch (error) {
    if (responseText) message = responseText
  }
  throw new Error(message)
}

export const downloadFileResponse = async (response, fallbackName) => {
  await assertFileResponse(response, '下载失败，请稍后重试。')

  const fileName = getDownloadFileName(
    response.headers?.['content-disposition'],
    fallbackName,
  )
  const objectUrl = URL.createObjectURL(response.data)
  const link = document.createElement('a')
  link.href = objectUrl
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(objectUrl)
}

export const delay = (milliseconds) => new Promise(resolve => setTimeout(resolve, milliseconds))
