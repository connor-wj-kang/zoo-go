const {request, mediaUrl} = require('../../../utils/api')

const MENU = [
    {icon: '🐾', label: '宠物档案', desc: '管理宠物档案'},
    {icon: '⌖', label: '地址管理', desc: '管理默认服务地址'},
    {icon: '◷', label: '订阅消息授权', desc: '订单节点提醒'},
    {icon: '◇', label: '隐私设置', desc: '门牌与授权记录'},
    {icon: '▤', label: '服务协议与退款规则', desc: ''},
    {icon: '☎', label: '联系客服', desc: '09:00–21:00'},
]

Page({
    data: {
        loading: true,
        profile: null,
        profileInitial: '宠',
        petCount: 0,
        menu: MENU,
    },

    onShow() {
        this.loadProfile()
    },

    onPullDownRefresh() {
        this.loadProfile().finally(() => wx.stopPullDownRefresh())
    },

    async loadProfile() {
        try {
            const home = await request('/v1/owner/home')
            const profile = {
                ...home.profile,
                avatar: mediaUrl(home.profile.avatar),
            }
            this.setData({
                loading: false,
                profile,
                profileInitial: (profile.name || '宠').slice(0, 1),
                petCount: (home.pets || []).length,
                menu: MENU.map((item) => item.label === '宠物档案' ? {...item, desc: `${(home.pets || []).length} 个宠物`} : item),
            })
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    switchStudent() {
        wx.reLaunch({url: '/pages/student/hall/index'})
    },

    goProfile() {
        wx.navigateTo({url: '/pages/profile/index'})
    },

    onMenu(event) {
        const label = event.currentTarget.dataset.label
        if (label === '联系客服') {
            wx.showModal({title: '联系客服', content: '客服在线时间 09:00–21:00，请通过平台客服入口提交问题。', showCancel: false})
            return
        }
        if (label === '订阅消息授权') {
            wx.requestSubscribeMessage({
                tmplIds: [],
                success: () => wx.showToast({title: '消息模板尚未配置', icon: 'none'}),
                fail: () => wx.showToast({title: '模板待配置', icon: 'none'}),
            })
            return
        }
        wx.showToast({title: '该设置项尚未开放', icon: 'none'})
    },
})
