const {request, q} = require('../../../utils/api')
const {decorateTask, decorateBid, fmtTime, tapFeedback} = require('../../../utils/format')

const requestPayment = (params) => new Promise((resolve, reject) => {
    wx.requestPayment({
        ...params,
        success: resolve,
        fail: (error) => {
            const message = error.errMsg || '微信支付未完成'
            if (message.includes('cancel')) {
                const cancelled = new Error('已取消支付')
                cancelled.cancelled = true
                reject(cancelled)
                return
            }
            reject(new Error(message))
        },
    })
})

Page({
    data: {
        id: '',
        loading: true,
        task: null,
        bids: [],
        selectedBid: null,
        showSheet: false,
        actionLoading: false,
        navTitle: '报价详情',
    },

    onLoad(options) {
        if (!options.id) {
            wx.showToast({title: '缺少任务编号', icon: 'none'})
            return
        }
        this.setData({id: options.id})
    },

    onShow() {
        if (this.data.id) {
            this.loadTask()
            this.startTimer()
        }
    },

    onHide() {
        this.stopTimer()
    },

    onUnload() {
        this.stopTimer()
    },

    async loadTask() {
        try {
            const task = await request(`/v1/tasks/${this.data.id}?${q({role: 'owner'})}`)
            if (!['BIDDING', 'PENDING_PAYMENT'].includes(task.status)) {
                wx.redirectTo({url: `/pages/owner/order/index?id=${task.orderId || task.id}`})
                return
            }
            const decorated = decorateTask(task)
            const bids = (task.bids || []).map(decorateBid).map((bid, index, list) => ({
                ...bid,
                lowest: index === 0 && list.length > 1,
            }))
            this.setData({
                loading: false,
                task: decorated,
                bids,
                navTitle: task.status === 'BIDDING' ? '报价详情' : '待付款',
            })
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    startTimer() {
        this.stopTimer()
        this.timer = setInterval(() => {
            const task = this.data.task
            if (!task) {
                return
            }
            const remainingSec = Math.max(0, task.remainingSec - 1)
            const payRemainingSec = Math.max(0, task.payRemainingSec - 1)
            this.setData({
                'task.remainingSec': remainingSec,
                'task.remainingText': fmtTime(remainingSec),
                'task.payRemainingSec': payRemainingSec,
                'task.payRemainingText': fmtTime(payRemainingSec),
            })
        }, 1000)
    },

    stopTimer() {
        if (this.timer) {
            clearInterval(this.timer)
            this.timer = null
        }
    },

    pickBid(event) {
        tapFeedback()
        const bid = this.data.bids.find((item) => item.id === event.currentTarget.dataset.id)
        this.setData({selectedBid: bid, showSheet: true})
    },

    closeSheet() {
        if (!this.data.actionLoading) {
            this.setData({showSheet: false})
        }
    },

    noop() {},

    async confirmAward() {
        if (this.data.actionLoading || !this.data.selectedBid) {
            return
        }
        this.setData({actionLoading: true})
        try {
            await request(`/v1/tasks/${this.data.id}/award`, {
                method: 'POST',
                data: {bidId: this.data.selectedBid.id},
            })
            this.setData({showSheet: false, selectedBid: null, actionLoading: false})
            wx.showToast({title: '已锁定，请完成支付', icon: 'none'})
            this.loadTask()
        } catch (error) {
            this.setData({actionLoading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    async pay() {
        const task = this.data.task
        if (!task || !task.orderId || this.data.actionLoading) {
            return
        }
        wx.showModal({
            title: '确认支付',
            content: `确认通过微信支付 ¥${task.priceText}？支付结果以微信回调为准。`,
            confirmText: '确认支付',
            confirmColor: '#EA580C',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                this.setData({actionLoading: true})
                try {
                    const challenge = await request(`/v1/orders/${task.orderId}/payments`, {
                        method: 'POST',
                        data: {},
                    })
                    await requestPayment(challenge.params)
                    const paymentResult = await this.waitForPayment(task.orderId, challenge.paymentId)
                    if (paymentResult === 'late') {
                        this.setData({actionLoading: false})
                        wx.showModal({
                            title: '支付已超过保留时间',
                            content: '款项已支付成功，但订单资格已超时。系统已自动转入原路退款审核，请勿重复支付。',
                            showCancel: false,
                        })
                        return
                    }
                    if (paymentResult !== 'paid') {
                        this.setData({actionLoading: false})
                        wx.showModal({
                            title: '支付结果确认中',
                            content: '微信已受理支付，但服务端尚未确认到账。请勿重复支付，稍后返回本页刷新。',
                            showCancel: false,
                        })
                        return
                    }
                    wx.showToast({title: '支付成功', icon: 'success'})
                    setTimeout(() => {
                        wx.redirectTo({url: `/pages/owner/order/index?id=${task.orderId}`})
                    }, 450)
                } catch (error) {
                    this.setData({actionLoading: false})
                    if (!error.cancelled) {
                        wx.showToast({title: error.message, icon: 'none'})
                    }
                }
            },
        })
    },

    async waitForPayment(orderId, paymentId) {
        for (let attempt = 0; attempt < 8; attempt += 1) {
            try {
                const task = await request(`/v1/orders/${orderId}/payments/${paymentId}`)
                if (task.order && task.order.status === 'PAID') {
                    return 'paid'
                }
                if (task.order && task.order.paymentStatus === 'PAID_LATE') {
                    return 'late'
                }
            } catch (error) {
                if (error.statusCode && error.statusCode < 500) {
                    throw error
                }
            }
            await new Promise((resolve) => setTimeout(resolve, 1000))
        }
        return false
    },

    async reopen() {
        this.setData({actionLoading: true})
        try {
            await request(`/v1/tasks/${this.data.id}/reopen`, {
                method: 'POST',
                data: {},
            })
            wx.showToast({title: '已重新开放竞价', icon: 'none'})
            await this.loadTask()
            this.setData({actionLoading: false})
            this.startTimer()
        } catch (error) {
            this.setData({actionLoading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    cancelTask() {
        wx.showModal({
            title: '撤销任务',
            content: '撤销后所有报价自动失效；当前未付款，无需退款。',
            confirmText: '确认撤单',
            confirmColor: '#EA580C',
            success: async (result) => {
                if (!result.confirm) {
                    return
                }
                try {
                    await request(`/v1/tasks/${this.data.id}/cancel`, {
                        method: 'POST',
                        data: {},
                    })
                    wx.showToast({title: '任务已撤单', icon: 'none'})
                    setTimeout(() => wx.reLaunch({url: '/pages/owner/home/index'}), 400)
                } catch (error) {
                    wx.showToast({title: error.message, icon: 'none'})
                }
            },
        })
    },

    goStudentHall() {
        wx.navigateTo({url: `/pages/student/task/index?id=${this.data.id}`})
    },
})
