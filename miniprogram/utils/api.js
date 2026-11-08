const unwrap = (body) => body && Object.prototype.hasOwnProperty.call(body, 'data') ? body.data : body

const errorFromResponse = (response) => {
    const body = response.data || {}
    const payload = typeof body === 'string' ? {} : body
    const message = (payload.error && payload.error.message) || `请求失败（${response.statusCode}）`
    const error = new Error(message)
    error.code = payload.error && payload.error.code
    error.statusCode = response.statusCode
    return error
}

const send = (path, options, token) => new Promise((resolve, reject) => {
    const app = getApp()
    const baseUrl = app.globalData.apiBase.replace(/\/$/, '')
    wx.request({
        url: `${baseUrl}${path}`,
        method: options.method || 'GET',
        data: options.data,
        timeout: options.timeout || 15000,
        header: {
            'content-type': 'application/json',
            ...(token ? {Authorization: `Bearer ${token}`} : {}),
            ...(options.header || {}),
        },
        success: (response) => {
            if (response.statusCode >= 200 && response.statusCode < 300) {
                resolve(unwrap(response.data))
                return
            }
            reject(errorFromResponse(response))
        },
        fail: (error) => reject(new Error(error.errMsg || '无法连接后端服务，请确认 Go API 已启动')),
    })
})

const request = async (path, options = {}) => {
    const app = getApp()
    let token = options.auth === false ? '' : await app.ensureAuth()
    try {
        return await send(path, options, token)
    } catch (error) {
        if (error.statusCode !== 401 || options.auth === false || options.retried) {
            throw error
        }
        app.clearSession()
        token = await app.ensureAuth(true)
        return send(path, {...options, retried: true}, token)
    }
}

const upload = async (filePath) => {
    const app = getApp()
    const baseUrl = app.globalData.apiBase.replace(/\/$/, '')
    const uploadOnce = (token) => new Promise((resolve, reject) => {
        wx.uploadFile({
            url: `${baseUrl}/v1/uploads`,
            filePath,
            name: 'file',
            timeout: 30000,
            header: {Authorization: `Bearer ${token}`},
            success: (response) => {
                let body = response.data
                if (typeof body === 'string') {
                    try {
                        body = JSON.parse(body)
                    } catch (error) {
                        reject(new Error('上传响应格式不正确'))
                        return
                    }
                }
                if (response.statusCode >= 200 && response.statusCode < 300) {
                    resolve(unwrap(body))
                    return
                }
                reject(errorFromResponse({...response, data: body}))
            },
            fail: (error) => reject(new Error(error.errMsg || '图片上传失败')),
        })
    })
    let token = await app.ensureAuth()
    try {
        return await uploadOnce(token)
    } catch (error) {
        if (error.statusCode !== 401) {
            throw error
        }
        app.clearSession()
        token = await app.ensureAuth(true)
        return uploadOnce(token)
    }
}

const q = (params) => Object.keys(params)
    .filter((key) => params[key] !== undefined && params[key] !== null && params[key] !== '')
    .map((key) => `${encodeURIComponent(key)}=${encodeURIComponent(params[key])}`)
    .join('&')

module.exports = {
    request,
    upload,
    q,
    apiBase: () => getApp().globalData.apiBase,
    mediaUrl: (path) => {
        if (!path) {
            return ''
        }
        if (/^https?:\/\//i.test(path) || path.startsWith('wxfile://')) {
            return path
        }
        if (path.startsWith('/uploads/')) {
            return `${getApp().globalData.apiBase.replace(/\/$/, '')}${path}`
        }
        return path
    },
}
