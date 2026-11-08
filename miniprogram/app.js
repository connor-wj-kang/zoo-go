const TOKEN_KEY = 'zoo.authToken'
const PROFILE_KEY = 'zoo.authProfile'
const DEVICE_KEY = 'zoo.deviceId'

const loginCode = () => new Promise((resolve) => {
    wx.login({
        success: (result) => resolve(result.code || ''),
        fail: () => resolve(''),
    })
})

const requestLogin = (url, data) => new Promise((resolve, reject) => {
    wx.request({
        url,
        method: 'POST',
        data,
        timeout: 10000,
        header: {'content-type': 'application/json'},
        success: (response) => {
            const body = response.data || {}
            if (response.statusCode >= 200 && response.statusCode < 300 && body.data && body.data.token) {
                resolve(body.data)
                return
            }
            const message = (body.error && body.error.message) || `登录失败（${response.statusCode}）`
            const error = new Error(message)
            error.code = body.error && body.error.code
            error.statusCode = response.statusCode
            reject(error)
        },
        fail: (error) => reject(new Error(error.errMsg || '无法连接后端服务，请确认 Go API 已启动')),
    })
})

const deviceID = () => {
    let value = wx.getStorageSync(DEVICE_KEY)
    if (!value) {
        value = `wx_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 12)}`
        wx.setStorageSync(DEVICE_KEY, value)
    }
    return value
}

App({
    globalData: {
        apiBase: wx.getStorageSync('apiBase') || 'http://127.0.0.1:8080',
        token: '',
        profile: null,
        authReady: null,
        authError: '',
    },

    onLaunch() {
        this.globalData.token = wx.getStorageSync(TOKEN_KEY) || ''
        this.globalData.profile = wx.getStorageSync(PROFILE_KEY) || null
        this.globalData.authReady = this.ensureAuth().catch((error) => {
            this.globalData.authError = error.message
            return ''
        })
    },

    ensureAuth(force = false) {
        if (!force && this.globalData.token) {
            return Promise.resolve(this.globalData.token)
        }
        if (this.authPromise) {
            return this.authPromise
        }
        this.authPromise = this.login().finally(() => {
            this.authPromise = null
        })
        return this.authPromise
    },

    async login() {
        const code = await loginCode()
        const baseUrl = this.globalData.apiBase.replace(/\/$/, '')
        const session = await requestLogin(`${baseUrl}/v1/auth/wechat/login`, {
            code,
            deviceId: deviceID(),
            nickname: '',
            avatar: '',
        })
        this.setSession(session)
        return session.token
    },

    setSession(session) {
        this.globalData.token = session.token
        this.globalData.profile = session.profile || null
        this.globalData.authError = ''
        wx.setStorageSync(TOKEN_KEY, session.token)
        wx.setStorageSync(PROFILE_KEY, session.profile || null)
    },

    setProfile(profile) {
        this.globalData.profile = profile
        wx.setStorageSync(PROFILE_KEY, profile)
    },

    clearSession() {
        this.globalData.token = ''
        this.globalData.profile = null
        wx.removeStorageSync(TOKEN_KEY)
    },

    async logout() {
        const token = this.globalData.token
        this.clearSession()
        if (!token) {
            return
        }
        const baseUrl = this.globalData.apiBase.replace(/\/$/, '')
        await new Promise((resolve) => {
            wx.request({
                url: `${baseUrl}/v1/auth/logout`,
                method: 'POST',
                header: {Authorization: `Bearer ${token}`, 'content-type': 'application/json'},
                complete: resolve,
            })
        })
    },
})
