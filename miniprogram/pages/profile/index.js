const {request, upload, mediaUrl} = require('../../utils/api')

Page({
    data: {
        loading: true,
        saving: false,
        name: '',
        avatar: '',
        avatarPreview: '',
        avatarChanged: false,
        initials: '宠',
    },

    onLoad() {
        this.loadProfile()
    },

    async loadProfile() {
        try {
            const profile = await request('/v1/auth/me')
            this.setData({
                loading: false,
                name: profile.name || '',
                avatar: profile.avatar || '',
                avatarPreview: mediaUrl(profile.avatar),
                initials: (profile.name || '宠').slice(0, 1),
            })
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    onNameInput(event) {
        const name = event.detail.value
        this.setData({
            name,
            initials: (name.trim() || '宠').slice(0, 1),
        })
    },

    onChooseAvatar(event) {
        const avatarPreview = event.detail.avatarUrl
        if (!avatarPreview) {
            return
        }
        this.setData({
            avatarPreview,
            avatarChanged: true,
        })
    },

    async save() {
        if (this.data.saving) {
            return
        }
        const name = this.data.name.trim()
        if (!name) {
            wx.showToast({title: '请填写昵称', icon: 'none'})
            return
        }
        this.setData({saving: true})
        try {
            let avatar = this.data.avatar
            if (this.data.avatarChanged) {
                avatar = (await upload(this.data.avatarPreview)).url
            }
            const profile = await request('/v1/profile', {
                method: 'POST',
                data: {name, avatar},
            })
            getApp().setProfile(profile)
            wx.showToast({title: '资料已更新', icon: 'success'})
            setTimeout(() => wx.navigateBack(), 450)
        } catch (error) {
            wx.showToast({title: error.message, icon: 'none'})
            this.setData({saving: false})
        }
    },
})
