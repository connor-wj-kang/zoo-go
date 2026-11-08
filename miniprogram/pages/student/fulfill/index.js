const {request, q, upload} = require('../../../utils/api')
const {decorateTask, tapFeedback} = require('../../../utils/format')

Page({
    data: {
        id: '',
        loading: true,
        actionLoading: false,
        task: null,
        photos: [],
    },

    onLoad(options) {
        if (!options.id) {
            wx.showToast({title: '缺少订单编号', icon: 'none'})
            return
        }
        this.setData({id: options.id})
        this.loadOrder()
    },

    onShow() {
        if (this.data.task) {
            this.loadOrder(false)
        }
    },

    async loadOrder(showLoading = true) {
        if (showLoading) {
            this.setData({loading: true})
        }
        try {
            const task = await request(`/v1/orders/${this.data.id}?${q({role: 'student'})}`)
            this.setData({loading: false, task: decorateTask(task)})
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    checkIn() {
        if (this.data.actionLoading) {
            return
        }
        wx.showModal({
            title: '到场签到',
            content: '请确认你已到达服务地址并准备开始服务。签到时间将记录在订单中。',
            confirmText: '确认签到',
            confirmColor: '#0D9488',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                this.setData({actionLoading: true})
                try {
                    await request(`/v1/orders/${this.data.id}/checkin`, {
                        method: 'POST',
                        data: {},
                    })
                    wx.showToast({title: '签到成功', icon: 'success'})
                    await this.loadOrder(false)
                } catch (error) {
                    wx.showToast({title: error.message, icon: 'none'})
                } finally {
                    this.setData({actionLoading: false})
                }
            },
        })
    },

    choosePhotos() {
        const remaining = Math.max(0, 4 - this.data.photos.length)
        if (!remaining) {
            wx.showToast({title: '最多上传 4 张凭证', icon: 'none'})
            return
        }
        wx.chooseMedia({
            count: remaining,
            mediaType: ['image'],
            sourceType: ['camera', 'album'],
            sizeType: ['compressed'],
            success: (result) => {
                const paths = result.tempFiles.map((file) => file.tempFilePath)
                this.setData({photos: this.data.photos.concat(paths)})
            },
        })
    },

    removePhoto(event) {
        const index = Number(event.currentTarget.dataset.index)
        const photos = this.data.photos.slice()
        photos.splice(index, 1)
        this.setData({photos})
    },

    previewPhoto(event) {
        wx.previewImage({
            current: event.currentTarget.dataset.src,
            urls: this.data.photos,
        })
    },

    previewEvidence(event) {
        wx.previewImage({
            current: event.currentTarget.dataset.src,
            urls: this.data.task.photos,
        })
    },

    complete() {
        if (this.data.actionLoading) {
            return
        }
        if (this.data.photos.length < 2) {
            wx.showToast({title: '请至少上传 2 张服务凭证', icon: 'none'})
            return
        }
        tapFeedback()
        wx.showModal({
            title: '提交完工',
            content: '请确认照护清单已完成、照片不包含他人面部/证件/密码。提交后等待主人验收。',
            confirmText: '提交完工',
            confirmColor: '#0D9488',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                this.setData({actionLoading: true})
                try {
                    const photos = []
                    for (const path of this.data.photos) {
                        photos.push((await upload(path)).url)
                    }
                    await request(`/v1/orders/${this.data.id}/complete`, {
                        method: 'POST',
                        data: {photos},
                    })
                    wx.showToast({title: '完工凭证已提交', icon: 'success'})
                    this.setData({photos: []})
                    await this.loadOrder(false)
                } catch (error) {
                    wx.showToast({title: error.message, icon: 'none'})
                } finally {
                    this.setData({actionLoading: false})
                }
            },
        })
    },

    callOwner() {
        wx.showModal({
            title: '联系主人',
            content: '正式环境将通过平台虚拟号联系主人，并记录通话留痕。',
            showCancel: false,
        })
    },

    reportIssue() {
        wx.showModal({
            title: '上报异常',
            content: '如遇宠物受伤、无法进门、财物争议等情况，请立即停止服务并通过平台客服上报。',
            showCancel: false,
        })
    },
})
