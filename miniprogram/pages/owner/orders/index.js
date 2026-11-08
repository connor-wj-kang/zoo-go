const {request, q} = require('../../../utils/api')
const {decorateTask} = require('../../../utils/format')

const FILTERS = [
    {value: 'all', label: '全部'},
    {value: 'bidding', label: '竞价中'},
    {value: 'fulfilling', label: '履约中'},
    {value: 'completed', label: '已完成'},
]

Page({
    data: {
        filters: FILTERS.map((item, index) => ({...item, selected: index === 0})),
        filter: 'all',
        tasks: [],
        loading: true,
        error: '',
    },

    onLoad() {
        this.loadTasks()
    },

    onShow() {
        if (!this.data.loading) {
            this.loadTasks(false)
        }
    },

    onPullDownRefresh() {
        this.loadTasks(false).finally(() => wx.stopPullDownRefresh())
    },

    async loadTasks(showLoading = true) {
        if (showLoading) {
            this.setData({loading: true, error: ''})
        }
        try {
            const tasks = await request(`/v1/owner/tasks?${q({filter: this.data.filter})}`)
            this.setData({loading: false, error: '', tasks: (tasks || []).map(decorateTask)})
        } catch (error) {
            this.setData({loading: false, error: error.message})
        }
    },

    selectFilter(event) {
        const filter = event.currentTarget.dataset.value
        this.setData({
            filter,
            filters: FILTERS.map((item) => ({...item, selected: item.value === filter})),
        })
        this.loadTasks()
    },

    goTask(event) {
        const {id, status, orderId} = event.currentTarget.dataset
        if (['BIDDING', 'PENDING_PAYMENT'].includes(status)) {
            wx.navigateTo({url: `/pages/owner/task/index?id=${id}`})
            return
        }
        wx.navigateTo({url: `/pages/owner/order/index?id=${orderId}`})
    },

    goPublish() {
        wx.navigateTo({url: '/pages/owner/publish/index'})
    },

    retry() {
        this.loadTasks()
    },
})
