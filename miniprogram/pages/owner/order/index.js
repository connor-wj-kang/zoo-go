const {request, q} = require('../../../utils/api')
const {decorateTask, tapFeedback} = require('../../../utils/format')

const TIMELINE = [
    {key: 'BOOKED', label: '已预约'},
    {key: 'IN_SERVICE', label: '服务中'},
    {key: 'PENDING_CONFIRM', label: '待验收'},
    {key: 'COMPLETED', label: '已完成'},
]

Page({
    data: {
        id: '',
        loading: true,
        actionLoading: false,
        task: null,
        timeline: [],
        showRating: false,
        stars: 5,
        starOptions: [1, 2, 3, 4, 5].map((value) => ({value})),
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
            const task = await request(`/v1/orders/${this.data.id}?${q({role: 'owner'})}`)
            if (['BIDDING', 'PENDING_PAYMENT'].includes(task.status)) {
                wx.redirectTo({url: `/pages/owner/task/index?id=${task.id}`})
                return
            }
            const decorated = decorateTask(task)
            const currentIndex = Math.max(0, TIMELINE.findIndex((item) => item.key === task.status))
            const timeline = TIMELINE.map((item, index) => ({
                ...item,
                done: index <= currentIndex,
                current: index === currentIndex,
                isLast: index === TIMELINE.length - 1,
            }))
            this.setData({loading: false, task: decorated, timeline})
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    callStudent() {
        wx.showModal({
            title: '平台虚拟号',
            content: '为保护双方隐私，请通过平台客服转联系服务学生。',
            showCancel: false,
        })
    },

    copyAddress() {
        if (!this.data.task || !this.data.task.addressDetail) {
            return
        }
        wx.setClipboardData({data: this.data.task.addressDetail})
    },

    previewPhoto(event) {
        const current = event.currentTarget.dataset.src
        wx.previewImage({current, urls: this.data.task.photos})
    },

    confirmOrder() {
        const task = this.data.task
        if (!task || this.data.actionLoading) {
            return
        }
        wx.showModal({
            title: '确认验收',
            content: '确认服务已完成且凭证无误？验收后订单将完成。',
            confirmText: '确认验收',
            confirmColor: '#EA580C',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                this.setData({actionLoading: true})
                try {
                    await request(`/v1/orders/${this.data.id}/confirm`, {
                        method: 'POST',
                        data: {},
                    })
                    wx.showToast({title: '验收完成', icon: 'success'})
                    await this.loadOrder(false)
                } catch (error) {
                    wx.showToast({title: error.message, icon: 'none'})
                } finally {
                    this.setData({actionLoading: false})
                }
            },
        })
    },

    openRating() {
        tapFeedback()
        this.setData({showRating: true})
    },

    closeRating() {
        this.setData({showRating: false})
    },

    noop() {},

    selectStar(event) {
        this.setData({stars: Number(event.currentTarget.dataset.value)})
    },

    async submitRating() {
        if (this.data.actionLoading) {
            return
        }
        this.setData({actionLoading: true})
        try {
            await request(`/v1/orders/${this.data.id}/rate`, {
                method: 'POST',
                data: {stars: this.data.stars},
            })
            this.setData({showRating: false})
            wx.showToast({title: '评价成功', icon: 'success'})
            await this.loadOrder(false)
        } catch (error) {
            wx.showToast({title: error.message, icon: 'none'})
        } finally {
            this.setData({actionLoading: false})
        }
    },

    requestRefund() {
        const task = this.data.task
        if (!task || !task.canRefund || this.data.actionLoading) {
            return
        }
        wx.showModal({
            title: '申请退款',
            content: '退款提交后进入人工审核，审核通过后原路退回微信支付账户。',
            editable: true,
            placeholderText: '请填写退款原因（选填）',
            confirmText: '提交申请',
            confirmColor: '#EA580C',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                this.setData({actionLoading: true})
                try {
                    await request(`/v1/orders/${this.data.id}/refunds`, {
                        method: 'POST',
                        data: {reason: (result.content || '').trim() || '主人申请取消服务并退款'},
                    })
                    wx.showToast({title: '退款申请已提交', icon: 'success'})
                    await this.loadOrder(false)
                } catch (error) {
                    wx.showToast({title: error.message, icon: 'none'})
                } finally {
                    this.setData({actionLoading: false})
                }
            },
        })
    },

    contactSupport() {
        wx.showModal({
            title: '联系客服',
            content: '已为你打开售后咨询入口，请通过平台客服提交订单号与问题说明。',
            showCancel: false,
        })
    },

    rebook() {
        wx.navigateTo({url: '/pages/owner/publish/index'})
    },
})
