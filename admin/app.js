const TOKEN_KEY = 'zoo.adminToken'

const VIEW_META = {
    overview: {eyebrow: '运营工作台', title: '运营概览', subtitle: '查看平台实时状态与待处理事项。'},
    dispatch: {eyebrow: '智能调度', title: '5km 派单对比', subtitle: '用真实待派任务对比贪心与 KM（匈牙利算法）的批量匹配结果。'},
    verifications: {eyebrow: '准入管理', title: '学籍审核', subtitle: '核验学生身份与在校状态，控制平台报价准入。'},
    refunds: {eyebrow: '资金处置', title: '退款审核', subtitle: '处理主人退款申请，并跟进微信原路退款结果。'},
    users: {eyebrow: '人员管理', title: '用户管理', subtitle: '查看主人和学生账号，处理停用与恢复。'},
    orders: {eyebrow: '履约管理', title: '订单管理', subtitle: '追踪订单状态、双方人员、服务时间与金额。'},
    audits: {eyebrow: '风控留痕', title: '审计日志', subtitle: '查询审核、退款和账号处置的后台操作记录。'},
}

const state = {
    token: '',
    view: 'overview',
    filters: {
        verifications: 'PENDING',
        refunds: 'REQUESTED',
        users: {status: 'ALL', query: ''},
        orders: 'ALL',
    },
    data: {},
    modalResolve: null,
}

const root = document.getElementById('view-root')
const loginScreen = document.getElementById('login-screen')
const appShell = document.getElementById('app-shell')
const loginForm = document.getElementById('login-form')
const loginError = document.getElementById('login-error')
const loginButton = document.getElementById('login-button')
const toast = document.getElementById('toast')
const modalBackdrop = document.getElementById('modal-backdrop')
const modalForm = document.getElementById('modal-form')
const modalFieldWrap = document.getElementById('modal-field-wrap')
const modalField = document.getElementById('modal-field')
const modalTextarea = document.getElementById('modal-textarea')

const escapeHTML = (value) => String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;')

const money = (fen) => `¥${(Number(fen || 0) / 100).toFixed(2)}`

const formatDate = (value, withTime = true) => {
    if (!value) {
        return '—'
    }
    const date = new Date(value)
    if (!Number.isFinite(date.getTime())) {
        return '—'
    }
    return new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        ...(withTime ? {hour: '2-digit', minute: '2-digit', hour12: false} : {}),
    }).format(date)
}

const localDateInput = (date) => {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
}

const statusMap = {
    PENDING: ['待审核', 'amber'],
    VERIFIED: ['已通过', 'green'],
    REJECTED: ['已驳回', 'red'],
    EXPIRED: ['已过期', 'gray'],
    UNVERIFIED: ['未提交', 'gray'],
    ACTIVE: ['正常', 'green'],
    DISABLED: ['已停用', 'red'],
    REQUESTED: ['待处理', 'amber'],
    PROCESSING: ['处理中', 'blue'],
    SUCCESS: ['已完成', 'green'],
    FAILED: ['失败', 'red'],
    UNPAID: ['待付款', 'amber'],
    PAID: ['已支付', 'green'],
    IN_SERVICE: ['服务中', 'blue'],
    PENDING_CONFIRM: ['待验收', 'blue'],
    COMPLETED: ['已完成', 'green'],
    REFUNDING: ['退款中', 'amber'],
    REFUNDED: ['已退款', 'gray'],
    CANCELLED: ['已取消', 'gray'],
    PAID_LATE: ['迟到支付', 'red'],
    PREPARING: ['支付准备', 'amber'],
    PREPARED: ['待支付', 'amber'],
    CLOSED: ['已关闭', 'gray'],
}

const badge = (status, fallback) => {
    const [label, tone] = statusMap[status] || [fallback || status || '未知', 'gray']
    return `<span class="badge badge-${tone}">${escapeHTML(label)}</span>`
}

const avatar = (person) => {
    if (person.avatar) {
        return `<div class="avatar"><img src="${escapeHTML(person.avatar)}" alt=""></div>`
    }
    return `<div class="avatar">${escapeHTML((person.name || '用').slice(0, 1))}</div>`
}

const person = (name, subtitle, personAvatar = '') => `
    <div class="person-cell">
        ${avatar({name, avatar: personAvatar})}
        <div><strong>${escapeHTML(name || '—')}</strong><small>${escapeHTML(subtitle || '—')}</small></div>
    </div>
`

const showToast = (message, tone = 'success') => {
    toast.textContent = message
    toast.className = `toast ${tone === 'error' ? 'error' : ''}`
    toast.hidden = false
    clearTimeout(showToast.timer)
    showToast.timer = setTimeout(() => {
        toast.hidden = true
    }, 3200)
}

const api = async (path, options = {}) => {
    const response = await fetch(path, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            'X-Admin-Token': state.token,
            ...(options.headers || {}),
        },
    })
    let body = null
    try {
        body = await response.json()
    } catch (error) {
        body = null
    }
    if (!response.ok) {
        const message = body?.error?.message || `请求失败（${response.status}）`
        const requestError = new Error(message)
        requestError.status = response.status
        requestError.code = body?.error?.code
        if (response.status === 401 || (response.status === 403 && requestError.code === 'FORBIDDEN')) {
            requestError.auth = true
        }
        throw requestError
    }
    return body?.data
}

const setLoading = () => {
    root.innerHTML = '<div class="loading-state">正在读取运营数据…</div>'
}

const setError = (message) => {
    root.innerHTML = `<div class="error-state">${escapeHTML(message)}</div>`
}

const emptyState = (title, description) => `
    <div class="table-wrap"><div class="empty-state">
        <div class="empty-icon">✓</div>
        <h3>${escapeHTML(title)}</h3>
        <p>${escapeHTML(description)}</p>
    </div></div>
`

const filterButtons = (view, current, options) => `
    <div class="filter-group">
        ${options.map(([value, label]) => `
            <button class="filter-button ${current === value ? 'active' : ''}"
                type="button" data-filter="${view}" data-filter-value="${value}">${label}</button>
        `).join('')}
    </div>
`

const updateHeader = () => {
    const meta = VIEW_META[state.view]
    document.getElementById('page-eyebrow').textContent = meta.eyebrow
    document.getElementById('page-title').textContent = meta.title
    document.getElementById('page-subtitle').textContent = meta.subtitle
    document.querySelectorAll('.nav-item').forEach((item) => {
        item.classList.toggle('active', item.dataset.view === state.view)
    })
}

const renderOverview = (overview) => {
    root.innerHTML = `
        <div class="stats-grid">
            <article class="stat-card">
                <div class="stat-label">进行中订单</div>
                <div class="stat-value">${overview.activeOrders}</div>
                <div class="stat-note">待付款、已预约、服务中与待验收</div>
            </article>
            <article class="stat-card accent">
                <div class="stat-label">待审学籍</div>
                <div class="stat-value">${overview.pendingVerifications}</div>
                <div class="stat-note">审核通过前不能提交新报价</div>
            </article>
            <article class="stat-card accent">
                <div class="stat-label">待审退款</div>
                <div class="stat-value">${overview.pendingRefunds}</div>
                <div class="stat-note">需按退款规则人工处置</div>
            </article>
            <article class="stat-card">
                <div class="stat-label">平台用户</div>
                <div class="stat-value">${overview.users}</div>
                <div class="stat-note">${overview.disabledUsers} 个账号已停用</div>
            </article>
        </div>
        <div class="overview-grid">
            <section class="panel">
                <div class="panel-head">
                    <div><h2 class="panel-title">待办队列</h2><p class="panel-subtitle">优先处理影响交易与准入的事项</p></div>
                </div>
                <div class="queue-list">
                    <button class="queue-item" type="button" data-goto="verifications">
                        <span class="queue-icon">✓</span>
                        <span><strong>学籍资料审核</strong><small>核对学信网报告与申请资料</small></span>
                        <span class="queue-meta">${overview.pendingVerifications} 项待办</span>
                    </button>
                    <button class="queue-item" type="button" data-goto="refunds">
                        <span class="queue-icon">↩</span>
                        <span><strong>主人退款申请</strong><small>审核通过后调用微信原路退款</small></span>
                        <span class="queue-meta">${overview.pendingRefunds} 项待办</span>
                    </button>
                    <button class="queue-item" type="button" data-goto="users">
                        <span class="queue-icon">◎</span>
                        <span><strong>认证学生</strong><small>当前可用学生资质与账号状态</small></span>
                        <span class="queue-meta">${overview.verifiedStudents} 人有效</span>
                    </button>
                </div>
            </section>
            <section class="panel">
                <div class="panel-head">
                    <div><h2 class="panel-title">平台经营</h2><p class="panel-subtitle">按已支付及完成订单统计</p></div>
                </div>
                <div class="stats-grid" style="grid-template-columns:1fr 1fr">
                    <article class="stat-card" style="min-height:108px;padding:16px">
                        <div class="stat-label">成交金额</div>
                        <div class="stat-value" style="font-size:24px">${money(overview.grossPaidFen)}</div>
                    </article>
                    <article class="stat-card" style="min-height:108px;padding:16px">
                        <div class="stat-label">今日订单</div>
                        <div class="stat-value" style="font-size:24px">${overview.todayOrders}</div>
                    </article>
                </div>
                <div class="guide-list" style="margin-top:22px">
                    <div class="guide-item"><span class="guide-number">1</span><span>学籍审核前先核对在线验证码、姓名、学校与学号，驳回原因必须具体。</span></div>
                    <div class="guide-item"><span class="guide-number">2</span><span>退款只在服务开始前按规则处理，迟到支付退款不可驳回。</span></div>
                    <div class="guide-item"><span class="guide-number">3</span><span>停用账号会立即清除登录会话；恢复后用户可重新登录。</span></div>
                </div>
            </section>
        </div>
    `
}

const renderVerifications = (items) => {
    const toolbar = `
        <div class="toolbar">
            ${filterButtons('verifications', state.filters.verifications, [
                ['PENDING', '待审核'], ['VERIFIED', '已通过'], ['REJECTED', '已驳回'], ['EXPIRED', '已过期'], ['ALL', '全部'],
            ])}
            <div class="toolbar-spacer"></div>
            <span class="muted-cell">共 ${items.length} 条</span>
        </div>
    `
    if (!items.length) {
        root.innerHTML = toolbar + emptyState('当前没有待处理学籍资料', '新的学生认证申请会出现在这里。')
        return
    }
    root.innerHTML = toolbar + `
        <div class="table-wrap"><table>
            <thead><tr><th>申请学生</th><th>学校 / 学号</th><th>学信网验证码</th><th>提交时间</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>${items.map((item) => `
                <tr>
                    <td>${person(item.realName, item.userName, '')}</td>
                    <td><strong>${escapeHTML(item.school)}</strong><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.studentNo)} · ${escapeHTML(item.major || '专业未填')}</div></td>
                    <td><span class="mono">${escapeHTML(item.chsiCode)}</span><div class="muted-cell" style="margin-top:4px">入学 ${item.enrollmentYear}</div></td>
                    <td class="muted-cell">${formatDate(item.submittedAt)}</td>
                    <td>${badge(item.status)}${item.reviewNote ? `<div class="muted-cell" style="margin-top:5px;max-width:180px">${escapeHTML(item.reviewNote)}</div>` : ''}</td>
                    <td>${item.status === 'PENDING' ? `<div class="row-actions">
                        <button class="button button-soft" data-action="verification-approve" data-id="${escapeHTML(item.id)}">通过</button>
                        <button class="button button-danger" data-action="verification-reject" data-id="${escapeHTML(item.id)}">驳回</button>
                    </div>` : '<span class="muted-cell">已处理</span>'}</td>
                </tr>
            `).join('')}</tbody>
        </table></div>
    `
}

const dispatchAssignmentTable = (result) => {
    if (!result.assignments.length) {
        return emptyState('没有形成有效分配', '当前任务与学生之间没有 5 公里内的可用匹配。')
    }
    return `
        <div class="table-wrap"><table>
            <thead><tr><th>任务位置</th><th>匹配学生</th><th>直线距离</th><th>综合得分</th></tr></thead>
            <tbody>${result.assignments.map((item) => `
                <tr>
                    <td><span class="mono">${escapeHTML(item.taskId)}</span><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.taskCommunity)}</div></td>
                    <td><strong>${escapeHTML(item.studentName)}</strong><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.studentId)}</div></td>
                    <td><strong>${Number(item.distanceKm).toFixed(1)} km</strong></td>
                    <td><strong>${Number(item.score).toFixed(1)}</strong></td>
                </tr>
            `).join('')}</tbody>
        </table></div>
    `
}

const renderDispatch = (data) => {
    if (!data.taskCount || !data.studentCount || !data.candidateEdges) {
        root.innerHTML = `
            <div class="dispatch-note">
                <strong>暂无可比较的真实派单样本</strong>
                <span>需要同时存在带定位的竞价任务，以及已认证、已开启定位且在 5 公里内的学生。</span>
            </div>
            ${emptyState('没有待派任务或可用学生', '发布带地图定位的任务后，再由学生端授权定位即可生成对比数据。')}
        `
        return
    }
    const greedy = data.greedy
    const km = data.km
    const gap = Number(data.scoreGap || 0)
    root.innerHTML = `
        <div class="dispatch-note">
            <strong>只读算法预演，不会自动成交。</strong>
            <span>半径固定 5km；得分 = 距离 50% + 评分 25% + 经验 15% + 基础信用 10%。KM 以全局总分最大为目标，贪心按单条最优边依次抢占。</span>
        </div>
        <div class="dispatch-metrics">
            <article class="stat-card"><div class="stat-label">待派任务</div><div class="stat-value">${data.taskCount}</div><div class="stat-note">仅统计已设置地图位置的竞价中任务</div></article>
            <article class="stat-card"><div class="stat-label">可用学生</div><div class="stat-value">${data.studentCount}</div><div class="stat-note">已认证且最近上报过位置</div></article>
            <article class="stat-card"><div class="stat-label">5km 候选边</div><div class="stat-value">${data.candidateEdges}</div><div class="stat-note">任务与学生之间可匹配的组合数</div></article>
            <article class="stat-card accent"><div class="stat-label">KM 相比贪心</div><div class="stat-value">${gap > 0 ? '+' : ''}${gap.toFixed(1)}</div><div class="stat-note">综合得分差，越高说明全局优化收益越明显</div></article>
        </div>
        <div class="comparison-grid">
            <section class="panel strategy-panel">
                <div class="panel-head">
                    <div><h2 class="panel-title">KM · 匈牙利算法</h2><p class="panel-subtitle">全局最优的一对一匹配</p></div>
                    <span class="badge badge-green">推荐对比</span>
                </div>
                <div class="strategy-stats">
                    <span><strong>${km.assigned}</strong> 单已匹配</span>
                    <span><strong>${Number(km.totalScore).toFixed(1)}</strong> 总分</span>
                    <span><strong>${Number(km.totalDistanceKM).toFixed(1)}</strong> km</span>
                </div>
            </section>
            <section class="panel strategy-panel">
                <div class="panel-head">
                    <div><h2 class="panel-title">贪心算法</h2><p class="panel-subtitle">每轮选择当前最高分边</p></div>
                    <span class="badge badge-gray">基线</span>
                </div>
                <div class="strategy-stats">
                    <span><strong>${greedy.assigned}</strong> 单已匹配</span>
                    <span><strong>${Number(greedy.totalScore).toFixed(1)}</strong> 总分</span>
                    <span><strong>${Number(greedy.totalDistanceKM).toFixed(1)}</strong> km</span>
                </div>
            </section>
        </div>
        <section class="panel dispatch-result-panel">
            <div class="panel-head"><div><h2 class="panel-title">KM 分配明细</h2><p class="panel-subtitle">批量派单时应优先参考全局匹配结果</p></div></div>
            ${dispatchAssignmentTable(km)}
        </section>
        <section class="panel dispatch-result-panel">
            <div class="panel-head"><div><h2 class="panel-title">贪心分配明细</h2><p class="panel-subtitle">实现简单、响应快，但可能因局部最优牺牲全局收益</p></div></div>
            ${dispatchAssignmentTable(greedy)}
        </section>
    `
}

const renderRefunds = (items) => {
    const toolbar = `
        <div class="toolbar">
            ${filterButtons('refunds', state.filters.refunds, [
                ['REQUESTED', '待处理'], ['PROCESSING', '处理中'], ['SUCCESS', '已完成'], ['FAILED', '失败'], ['REJECTED', '已驳回'], ['ALL', '全部'],
            ])}
            <div class="toolbar-spacer"></div>
            <span class="muted-cell">共 ${items.length} 条</span>
        </div>
    `
    if (!items.length) {
        root.innerHTML = toolbar + emptyState('当前没有退款申请', '主人提交退款后会进入审核队列。')
        return
    }
    root.innerHTML = toolbar + `
        <div class="table-wrap"><table>
            <thead><tr><th>订单 / 退款单</th><th>金额</th><th>申请原因</th><th>申请时间</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>${items.map((item) => `
                <tr>
                    <td><span class="mono">${escapeHTML(item.orderId)}</span><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.outRefundNo)}</div></td>
                    <td><strong>${money(item.amountFen)}</strong><div class="muted-cell" style="margin-top:4px">原支付 ${money(item.totalFen)}</div></td>
                    <td class="muted-cell" style="max-width:240px">${escapeHTML(item.reason || '未填写')}</td>
                    <td class="muted-cell">${formatDate(item.createdAt)}</td>
                    <td>${badge(item.status)}</td>
                    <td>${item.status === 'REQUESTED' ? `<div class="row-actions">
                        <button class="button button-soft" data-action="refund-approve" data-id="${escapeHTML(item.id)}">通过退款</button>
                        <button class="button button-danger" data-action="refund-reject" data-id="${escapeHTML(item.id)}">驳回</button>
                    </div>` : '<span class="muted-cell">—</span>'}</td>
                </tr>
            `).join('')}</tbody>
        </table></div>
    `
}

const renderUsers = (items) => {
    const filter = state.filters.users
    const toolbar = `
        <form id="user-search-form" class="toolbar">
            <input id="user-search" class="text-input" value="${escapeHTML(filter.query)}" placeholder="搜索姓名、用户 ID、学校或手机号">
            ${filterButtons('users', filter.status, [['ALL', '全部'], ['ACTIVE', '正常'], ['DISABLED', '已停用']])}
            <div class="toolbar-spacer"></div>
            <button class="button button-ghost" type="submit">搜索</button>
        </form>
    `
    if (!items.length) {
        root.innerHTML = toolbar + emptyState('没有匹配的用户', '尝试更换搜索词或账号状态。')
        return
    }
    root.innerHTML = toolbar + `
        <div class="table-wrap"><table>
            <thead><tr><th>用户</th><th>身份与认证</th><th>订单</th><th>注册时间</th><th>账号状态</th><th>操作</th></tr></thead>
            <tbody>${items.map((item) => {
                const roles = [item.ownerVerified ? '认证主人' : '', item.studentVerified ? '认证学生' : ''].filter(Boolean).join(' / ') || '普通用户'
                return `
                    <tr>
                        <td>${person(item.name, item.phoneMask || item.id, item.avatar)}</td>
                        <td><strong>${escapeHTML(roles)}</strong><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.school || '未填写学校')} · ${escapeHTML(statusMap[item.studentStatus]?.[0] || item.studentStatus || '未认证')}</div></td>
                        <td><strong>${item.activeOrders}</strong> 个进行中<div class="muted-cell" style="margin-top:4px">完成 ${item.doneCount} 单</div></td>
                        <td class="muted-cell">${formatDate(item.createdAt)}</td>
                        <td>${badge(item.accountStatus)}${item.disabledReason ? `<div class="muted-cell" style="margin-top:5px;max-width:180px">${escapeHTML(item.disabledReason)}</div>` : ''}</td>
                        <td><button class="button ${item.accountStatus === 'DISABLED' ? 'button-soft' : 'button-danger'}"
                            data-action="${item.accountStatus === 'DISABLED' ? 'user-enable' : 'user-disable'}"
                            data-id="${escapeHTML(item.id)}" data-name="${escapeHTML(item.name)}">${item.accountStatus === 'DISABLED' ? '恢复账号' : '停用账号'}</button></td>
                    </tr>
                `
            }).join('')}</tbody>
        </table></div>
    `
}

const renderOrders = (items) => {
    const toolbar = `
        <div class="toolbar">
            ${filterButtons('orders', state.filters.orders, [
                ['ALL', '全部'], ['UNPAID', '待付款'], ['PAID', '已支付'], ['IN_SERVICE', '服务中'], ['PENDING_CONFIRM', '待验收'], ['COMPLETED', '已完成'], ['REFUNDING', '退款中'], ['CANCELLED', '已取消'],
            ])}
            <div class="toolbar-spacer"></div>
            <span class="muted-cell">共 ${items.length} 条</span>
        </div>
    `
    if (!items.length) {
        root.innerHTML = toolbar + emptyState('没有匹配的订单', '新成交订单会出现在这里。')
        return
    }
    root.innerHTML = toolbar + `
        <div class="table-wrap"><table>
            <thead><tr><th>订单</th><th>主人</th><th>学生</th><th>服务</th><th>金额</th><th>状态</th><th>创建时间</th></tr></thead>
            <tbody>${items.map((item) => `
                <tr>
                    <td><span class="mono">${escapeHTML(item.id)}</span><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.petName || '宠物未找到')}</div></td>
                    <td>${person(item.ownerName, item.ownerId)}</td>
                    <td>${person(item.studentName, item.studentId)}</td>
                    <td><strong>${escapeHTML(item.serviceDate)}</strong><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.timeWindow)} · ${escapeHTML(item.community)}</div></td>
                    <td><strong>${money(item.amountFen)}</strong><div class="muted-cell" style="margin-top:4px">平台 ${money(item.feeFen)}</div></td>
                    <td>${badge(item.status)}${item.paymentStatus ? `<div style="margin-top:5px">${badge(item.paymentStatus, item.paymentStatus)}</div>` : ''}</td>
                    <td class="muted-cell">${formatDate(item.createdAt)}</td>
                </tr>
            `).join('')}</tbody>
        </table></div>
    `
}

const auditActionLabels = {
    STUDENT_VERIFICATION_APPROVED: '学籍审核通过',
    STUDENT_VERIFICATION_REJECTED: '学籍审核驳回',
    REFUND_APPROVED: '退款审核通过',
    REFUND_REJECTED: '退款审核驳回',
    USER_DISABLED: '停用账号',
    USER_ENABLED: '恢复账号',
}

const renderAudits = (items) => {
    if (!items.length) {
        root.innerHTML = emptyState('暂无审计记录', '后台审核和账号处置操作会自动记录在这里。')
        return
    }
    root.innerHTML = `
        <div class="table-wrap"><table>
            <thead><tr><th>操作</th><th>操作人</th><th>资源</th><th>说明</th><th>时间</th></tr></thead>
            <tbody>${items.map((item) => `
                <tr>
                    <td><strong>${escapeHTML(auditActionLabels[item.action] || item.action)}</strong></td>
                    <td class="muted-cell">${escapeHTML(item.actor)}</td>
                    <td><span class="mono">${escapeHTML(item.resourceId)}</span><div class="muted-cell" style="margin-top:4px">${escapeHTML(item.resourceType)}</div></td>
                    <td class="muted-cell" style="max-width:360px">${escapeHTML(item.detail || '—')}</td>
                    <td class="muted-cell">${formatDate(item.createdAt)}</td>
                </tr>
            `).join('')}</tbody>
        </table></div>
    `
}

const loadView = async () => {
    updateHeader()
    setLoading()
    try {
        if (state.view === 'overview') {
            state.data.overview = await api('/v1/admin/overview')
            renderOverview(state.data.overview)
        } else if (state.view === 'dispatch') {
            state.data.dispatch = await api('/v1/admin/dispatch/compare?limit=50')
            renderDispatch(state.data.dispatch)
        } else if (state.view === 'verifications') {
            const status = state.filters.verifications
            state.data.verifications = await api(`/v1/admin/student-verifications?status=${encodeURIComponent(status)}`)
            renderVerifications(state.data.verifications)
        } else if (state.view === 'refunds') {
            const status = state.filters.refunds
            state.data.refunds = await api(`/v1/admin/refunds?status=${encodeURIComponent(status)}`)
            renderRefunds(state.data.refunds)
        } else if (state.view === 'users') {
            const filter = state.filters.users
            state.data.users = await api(`/v1/admin/users?status=${encodeURIComponent(filter.status)}&q=${encodeURIComponent(filter.query)}`)
            renderUsers(state.data.users)
        } else if (state.view === 'orders') {
            const status = state.filters.orders
            state.data.orders = await api(`/v1/admin/orders?status=${encodeURIComponent(status)}`)
            renderOrders(state.data.orders)
        } else if (state.view === 'audits') {
            state.data.audits = await api('/v1/admin/audit-logs?limit=200')
            renderAudits(state.data.audits)
        }
    } catch (error) {
        if (error.auth) {
            logout()
            loginError.textContent = '管理员令牌已失效，请重新登录。'
            loginError.hidden = false
            return
        }
        setError(error.message)
    }
}

const openModal = (options) => {
    return new Promise((resolve) => {
        state.modalResolve = resolve
        document.getElementById('modal-eyebrow').textContent = options.eyebrow || '审核操作'
        document.getElementById('modal-title').textContent = options.title || '确认操作'
        document.getElementById('modal-description').textContent = options.description || ''
        document.getElementById('modal-confirm').textContent = options.confirmText || '确认'
        document.getElementById('modal-confirm').className = `button ${options.tone === 'danger' ? 'button-danger' : 'button-primary'}`
        const hasField = Boolean(options.fieldLabel)
        modalFieldWrap.hidden = !hasField
        if (hasField) {
            const isTextarea = options.fieldType === 'textarea'
            document.getElementById('modal-field-label').textContent = options.fieldLabel
            document.getElementById('modal-field-label').htmlFor = isTextarea ? 'modal-textarea' : 'modal-field'
            modalField.hidden = isTextarea
            modalTextarea.hidden = !isTextarea
            const input = isTextarea ? modalTextarea : modalField
            input.type = options.fieldType === 'date' ? 'date' : 'text'
            input.value = options.fieldValue || ''
            input.placeholder = options.placeholder || ''
            input.required = options.required !== false
        }
        modalBackdrop.hidden = false
        setTimeout(() => {
            if (hasField) {
                (modalField.hidden ? modalTextarea : modalField).focus()
            } else {
                document.getElementById('modal-confirm').focus()
            }
        }, 20)
    })
}

const closeModal = (value = null) => {
    if (!state.modalResolve) {
        return
    }
    const resolve = state.modalResolve
    state.modalResolve = null
    modalBackdrop.hidden = true
    resolve(value)
}

const handleAction = async (button) => {
    const action = button.dataset.action
    const id = button.dataset.id
    const request = async (path, options) => {
        button.disabled = true
        try {
            await api(path, options)
            await loadView()
            showToast('操作已完成')
        } catch (error) {
            showToast(error.message, 'error')
        } finally {
            button.disabled = false
        }
    }

    if (action === 'verification-approve') {
        const defaultDate = new Date()
        defaultDate.setFullYear(defaultDate.getFullYear() + 1)
        const value = await openModal({
            title: '通过学籍审核',
            description: '请确认已在学信网核对姓名、学校、学号与报告有效期。',
            fieldLabel: '认证有效期至',
            fieldType: 'date',
            fieldValue: localDateInput(defaultDate),
            confirmText: '确认通过',
            tone: 'primary',
        })
        if (value === null) return
        await request(`/v1/admin/student-verifications/${encodeURIComponent(id)}/approve`, {
            method: 'POST', body: JSON.stringify({validUntil: value}),
        })
    }
    if (action === 'verification-reject') {
        const value = await openModal({
            title: '驳回学籍申请',
            description: '学生端会看到驳回原因，并可以补充资料后重新提交。',
            fieldLabel: '驳回原因',
            fieldType: 'textarea',
            placeholder: '例如：在线验证码已失效，请重新申请学籍报告',
            confirmText: '确认驳回',
            tone: 'danger',
        })
        if (value === null) return
        await request(`/v1/admin/student-verifications/${encodeURIComponent(id)}/reject`, {
            method: 'POST', body: JSON.stringify({reason: value}),
        })
    }
    if (action === 'refund-approve') {
        const value = await openModal({
            title: '通过退款申请',
            description: '确认后将立即调用微信原路退款。请先核对订单状态、退款金额和退款规则。',
            confirmText: '确认退款',
            tone: 'danger',
        })
        if (value === null) return
        await request(`/v1/admin/refunds/${encodeURIComponent(id)}/approve`, {method: 'POST'})
    }
    if (action === 'refund-reject') {
        const reason = await openModal({
            title: '驳回退款申请',
            description: '驳回原因会写入审计日志。请确认订单不符合退款条件。',
            fieldLabel: '驳回原因',
            fieldType: 'textarea',
            placeholder: '例如：服务已开始，不符合退款条件',
            confirmText: '确认驳回',
            tone: 'danger',
        })
        if (reason === null) return
        await request(`/v1/admin/refunds/${encodeURIComponent(id)}/reject`, {
            method: 'POST', body: JSON.stringify({reason}),
        })
    }
    if (action === 'user-disable') {
        const reason = await openModal({
            title: `停用「${button.dataset.name}」`,
            description: '停用后该用户当前登录会立即失效，无法继续登录和操作。',
            fieldLabel: '停用原因',
            fieldType: 'textarea',
            placeholder: '填写风控原因，最多 200 字',
            confirmText: '确认停用',
            tone: 'danger',
        })
        if (reason === null) return
        await request(`/v1/admin/users/${encodeURIComponent(id)}/status`, {
            method: 'POST', body: JSON.stringify({status: 'DISABLED', reason}),
        })
    }
    if (action === 'user-enable') {
        const value = await openModal({
            title: `恢复「${button.dataset.name}」`,
            description: '恢复后用户可重新登录；历史审核与订单记录不会改变。',
            confirmText: '确认恢复',
            tone: 'primary',
        })
        if (value === null) return
        await request(`/v1/admin/users/${encodeURIComponent(id)}/status`, {
            method: 'POST', body: JSON.stringify({status: 'ACTIVE'}),
        })
    }
}

const switchView = (view) => {
    if (!VIEW_META[view]) {
        return
    }
    state.view = view
    loadView()
}

document.querySelectorAll('.nav-item').forEach((button) => {
    button.addEventListener('click', () => switchView(button.dataset.view))
})

root.addEventListener('click', (event) => {
    const go = event.target.closest('[data-goto]')
    if (go) {
        switchView(go.dataset.goto)
        return
    }
    const filter = event.target.closest('[data-filter]')
    if (filter) {
        const view = filter.dataset.filter
        const value = filter.dataset.filterValue
        if (view === 'users') {
            state.filters.users.status = value
        } else {
            state.filters[view] = value
        }
        loadView()
        return
    }
    const action = event.target.closest('[data-action]')
    if (action) {
        handleAction(action)
    }
})

root.addEventListener('submit', (event) => {
    if (event.target.id !== 'user-search-form') {
        return
    }
    event.preventDefault()
    state.filters.users.query = document.getElementById('user-search').value.trim()
    loadView()
})

loginForm.addEventListener('submit', async (event) => {
    event.preventDefault()
    const token = document.getElementById('admin-token').value.trim()
    if (!token) {
        return
    }
    loginError.hidden = true
    loginButton.disabled = true
    loginButton.textContent = '正在验证…'
    state.token = token
    try {
        await api('/v1/admin/overview')
        sessionStorage.setItem(TOKEN_KEY, token)
        loginScreen.hidden = true
        appShell.hidden = false
        switchView('overview')
    } catch (error) {
        state.token = ''
        loginError.textContent = error.message
        loginError.hidden = false
    } finally {
        loginButton.disabled = false
        loginButton.textContent = '验证并进入'
    }
})

document.getElementById('logout-button').addEventListener('click', () => logout())
document.getElementById('refresh-button').addEventListener('click', () => loadView())

const logout = () => {
    state.token = ''
    sessionStorage.removeItem(TOKEN_KEY)
    appShell.hidden = true
    loginScreen.hidden = false
    document.getElementById('admin-token').value = ''
}

modalForm.addEventListener('submit', (event) => {
    event.preventDefault()
    const fieldVisible = !modalFieldWrap.hidden
    let value = true
    if (fieldVisible) {
        value = (modalField.hidden ? modalTextarea : modalField).value.trim()
        if (!value) {
            return
        }
    }
    closeModal(value)
})

document.getElementById('modal-cancel').addEventListener('click', () => closeModal(null))
modalBackdrop.addEventListener('click', (event) => {
    if (event.target === modalBackdrop) {
        closeModal(null)
    }
})
document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !modalBackdrop.hidden) {
        closeModal(null)
    }
})

const boot = () => {
    const saved = sessionStorage.getItem(TOKEN_KEY)
    if (!saved) {
        loginScreen.hidden = false
        appShell.hidden = true
        return
    }
    state.token = saved
    loginScreen.hidden = true
    appShell.hidden = false
    switchView('overview')
}

boot()
