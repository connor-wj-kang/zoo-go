const {request, mediaUrl} = require('../../../utils/api')
const {money} = require('../../../utils/format')

const MENU = [
    {icon: '⌖', label: '常驻区域与接单半径', desc: '设置接单范围'},
    {icon: '◷', label: '空闲时段设置', desc: '工作日 18:00 后 / 周末'},
    {icon: '✓', label: '服务承诺与履约规范', desc: '签到、清单、凭证'},
    {icon: '◇', label: '隐私与授权管理', desc: '身份材料访问留痕'},
    {icon: '☎', label: '违规申诉与客服', desc: '订单争议处理'},
]

const dateText = (value) => {
    if (!value) {
        return ''
    }
    const date = new Date(value)
    if (!Number.isFinite(date.getTime())) {
        return ''
    }
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

Page({
    data: {
        loading: true,
        student: null,
        availableText: '0',
        initial: '学',
        realNameInput: '',
        schoolInput: '',
        studentNoInput: '',
        majorInput: '',
        enrollmentYearInput: '',
        chsiCodeInput: '',
        agreed: false,
        isPending: false,
        isRejected: false,
        isExpired: false,
        canSubmit: false,
        reviewNote: '',
        expiresText: '',
        submitting: false,
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
            const [student, earnings] = await Promise.all([
                request('/v1/student/profile'),
                request('/v1/student/earnings'),
            ])
            this.setData({
                loading: false,
                student: {...student, avatar: mediaUrl(student.avatar)},
                initial: (student.name || '学').slice(0, 1),
                schoolInput: student.school || '',
                isPending: student.verificationStatus === 'PENDING',
                isRejected: student.verificationStatus === 'REJECTED',
                isExpired: student.verificationStatus === 'EXPIRED',
                canSubmit: !student.verified && student.verificationStatus !== 'PENDING',
                reviewNote: student.verificationNote || '',
                expiresText: dateText(student.verificationExpiresAt),
                availableText: money(earnings.availableFen),
                menu: MENU.map((item) => item.label === '常驻区域与接单半径'
                    ? {...item, desc: `马连洼街道 · ${student.serviceRadiusKm}km`}
                    : item),
            })
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    switchOwner() {
        wx.reLaunch({url: '/pages/owner/home/index'})
    },

    goProfile() {
        wx.navigateTo({url: '/pages/profile/index'})
    },

    onFieldInput(event) {
        const field = event.currentTarget.dataset.field
        if (field) {
            this.setData({[field]: event.detail.value})
        }
    },

    onAgreementChange(event) {
        this.setData({agreed: event.detail.value.length > 0})
    },

    showVerificationHelp() {
        wx.showModal({
            title: '获取学籍在线验证码',
            content: '登录学信网，申请《教育部学籍在线验证报告》。打开报告后，填写报告中的在线验证码，并确保报告在审核期间有效。',
            confirmText: '复制网址',
            success: (result) => {
                if (result.confirm) {
                    wx.setClipboardData({data: 'https://www.chsi.com.cn/xlcx/'})
                }
            },
        })
    },

    async submitVerification() {
        const realName = this.data.realNameInput.trim()
        const school = this.data.schoolInput.trim()
        const studentNo = this.data.studentNoInput.trim()
        const major = this.data.majorInput.trim()
        const chsiCode = this.data.chsiCodeInput.trim()
        const enrollmentYear = Number(this.data.enrollmentYearInput)
        if (realName.length < 2) {
            wx.showToast({title: '请填写真实姓名', icon: 'none'})
            return
        }
        if (!school) {
            wx.showToast({title: '请填写学校名称', icon: 'none'})
            return
        }
        if (studentNo.length < 4) {
            wx.showToast({title: '请填写有效学号', icon: 'none'})
            return
        }
        if (!Number.isInteger(enrollmentYear) || enrollmentYear < 1980 || enrollmentYear > new Date().getFullYear()) {
            wx.showToast({title: '请填写有效入学年份', icon: 'none'})
            return
        }
        if (chsiCode.length < 6) {
            wx.showToast({title: '请填写在线验证码', icon: 'none'})
            return
        }
        if (!this.data.agreed) {
            wx.showToast({title: '请确认资料真实并同意审核', icon: 'none'})
            return
        }
        if (this.data.submitting) {
            return
        }
        this.setData({submitting: true})
        try {
            await request('/v1/student/verification', {
                method: 'POST',
                data: {realName, school, studentNo, major, enrollmentYear, chsiCode},
            })
            wx.showToast({title: '已提交，等待审核', icon: 'success'})
            await this.loadProfile()
        } catch (error) {
            wx.showToast({title: error.message, icon: 'none'})
        } finally {
            this.setData({submitting: false})
        }
    },

    onMenu(event) {
        wx.showToast({title: '该设置项尚未开放', icon: 'none'})
    },
})
