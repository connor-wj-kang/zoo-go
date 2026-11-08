const {request, mediaUrl} = require('../../../utils/api')
const {decorateTask, decoratePost, tapFeedback} = require('../../../utils/format')

const STEP_ICONS = {edit: '✎', bid: '⚖', pay: '¥', camera: '◎'}
const GUARANTEE_ICONS = {shield: '✓', eye: '◉', snow: '❄'}

Page({
    data: {
        loading: true,
        error: '',
        profile: null,
        area: null,
        tasks: [],
        pets: [],
        steps: [],
        guarantees: [],
        community: {total: 0, latest: [], latestText: '还没有动态，来分享第一条养宠日常吧'},
    },

    onLoad() {
        this.loadHome()
    },

    onShow() {
        if (!this.data.loading) {
            this.loadHome(false)
        }
    },

    onPullDownRefresh() {
        this.loadHome(false).finally(() => wx.stopPullDownRefresh())
    },

    async loadHome(showLoading = true) {
        if (showLoading) {
            this.setData({loading: true, error: ''})
        }
        try {
            const home = await request('/v1/owner/home')
            const community = home.community || {total: 0, latest: []}
            const latest = (community.latest || []).map(decoratePost)
            const latestPost = latest[0]
            this.setData({
                loading: false,
                error: '',
                profile: home.profile,
                area: home.area,
                tasks: (home.tasks || []).map(decorateTask),
                pets: (home.pets || []).map((pet) => ({...pet, image: mediaUrl(pet.image)})),
                steps: (home.steps || []).map((item) => ({...item, glyph: STEP_ICONS[item.icon] || '•'})),
                guarantees: (home.guarantees || []).map((item) => ({...item, glyph: GUARANTEE_ICONS[item.icon] || '✓'})),
                community: {
                    ...community,
                    latest,
                    latestText: latestPost
                        ? (latestPost.content || `分享了 ${latestPost.images.length} 张图片`)
                        : '还没有动态，来分享第一条养宠日常吧',
                },
            })
        } catch (error) {
            this.setData({loading: false, error: error.message})
        }
    },

    goPublish() {
        tapFeedback()
        wx.navigateTo({url: '/pages/owner/publish/index'})
    },

    goOrders() {
        wx.reLaunch({url: '/pages/owner/orders/index'})
    },

    goTask(event) {
        const {id, orderId, status} = event.currentTarget.dataset
        const url = ['BIDDING', 'PENDING_PAYMENT'].includes(status)
            ? `/pages/owner/task/index?id=${id}`
            : `/pages/owner/order/index?id=${orderId}`
        wx.navigateTo({url})
    },

    goStudent() {
        wx.reLaunch({url: '/pages/student/hall/index'})
    },

    onBell() {
        wx.showToast({title: '暂无新消息', icon: 'none'})
    },

    addPet() {
        wx.navigateTo({url: '/pages/owner/pet-edit/index'})
    },

    goCommunity() {
        wx.navigateTo({url: '/pages/community/index?role=owner'})
    },

    goCommunityPublish() {
        wx.navigateTo({url: '/pages/community/index?role=owner&compose=1'})
    },

    retry() {
        this.loadHome()
    },
})
