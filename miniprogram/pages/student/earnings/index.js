const {request} = require('../../../utils/api')
const {money, STATUS_META} = require('../../../utils/format')

Page({
    data: {
        loading: true,
        totalText: '0',
        pendingText: '0',
        availableText: '0',
        completedOrders: 0,
        records: [],
    },

    onShow() {
        this.loadEarnings()
    },

    onPullDownRefresh() {
        this.loadEarnings().finally(() => wx.stopPullDownRefresh())
    },

    async loadEarnings() {
        try {
            const earnings = await request('/v1/student/earnings')
            const records = (earnings.records || []).map((item) => ({
                ...item,
                amountText: money(item.amountFen),
                feeText: money(item.feeFen),
                studentText: money(item.studentFen),
                statusText: (STATUS_META[item.status] || {text: item.status}).text,
            }))
            this.setData({
                loading: false,
                totalText: money(earnings.totalFen),
                pendingText: money(earnings.pendingFen),
                availableText: money(earnings.availableFen),
                completedOrders: earnings.completedOrders,
                records,
            })
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    goTasks() {
        wx.reLaunch({url: '/pages/student/tasks/index'})
    },

    contactFinance() {
        wx.showModal({
            title: '对账异常反馈',
            content: '如对收益有疑问，请提供订单号并通过平台客服发起对账。',
            showCancel: false,
        })
    },
})
