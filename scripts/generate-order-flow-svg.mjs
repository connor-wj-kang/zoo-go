import fs from 'node:fs'
import path from 'node:path'

const W = 1800
const H = 1250
const colors = {
  bg: '#F7F5F2',
  ink: '#172033',
  muted: '#64748B',
  line: '#334155',
  taskFill: '#EAF2FF',
  taskStroke: '#3B82F6',
  orderFill: '#FFF4E8',
  orderStroke: '#F97316',
  paymentFill: '#EAF8F0',
  paymentStroke: '#10B981',
  refundFill: '#FDECEC',
  refundStroke: '#EF4444',
  neutralFill: '#F1F5F9',
  neutralStroke: '#94A3B8',
}

const parts = []
const esc = (value) => String(value)
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')

function text(x, y, value, options = {}) {
  const {
    size = 13, weight = 500, fill = colors.ink, anchor = 'middle', letter = 0,
  } = options
  parts.push(`<text x="${x}" y="${y}" text-anchor="${anchor}" font-size="${size}" font-weight="${weight}" fill="${fill}" letter-spacing="${letter}">${esc(value)}</text>`)
}

function label(x, y, lines, options = {}) {
  const { size = 12, weight = 650, fill = colors.muted, anchor = 'middle' } = options
  const list = Array.isArray(lines) ? lines : [lines]
  list.forEach((line, index) => {
    const lineY = y + index * (size + 4)
    parts.push(`<text x="${x}" y="${lineY}" text-anchor="${anchor}" font-size="${size}" font-weight="${weight}" fill="${colors.bg}" stroke="${colors.bg}" stroke-width="5" stroke-linejoin="round">${esc(line)}</text>`)
    parts.push(`<text x="${x}" y="${lineY}" text-anchor="${anchor}" font-size="${size}" font-weight="${weight}" fill="${fill}">${esc(line)}</text>`)
  })
}

function stateBox(x, y, w, h, title, subtitle, options = {}) {
  const {
    fill = colors.taskFill, stroke = colors.taskStroke, tag = '', tagFill = stroke,
  } = options
  parts.push(`<g filter="url(#softShadow)"><rect x="${x}" y="${y}" width="${w}" height="${h}" rx="18" fill="${fill}" stroke="${stroke}" stroke-width="2.2"/></g>`)
  if (tag) {
    const tagWidth = Math.max(76, tag.length * 12 + 24)
    parts.push(`<rect x="${x + w - tagWidth - 12}" y="${y + 12}" width="${tagWidth}" height="24" rx="12" fill="${tagFill}" opacity="0.14"/>`)
    text(x + w - tagWidth / 2 - 12, y + 29, tag, { size: 11, weight: 750, fill: tagFill })
  }
  const titleY = y + (subtitle ? 43 : h / 2 + 6)
  text(x + w / 2, titleY, title, { size: 17, weight: 800, letter: 0.2 })
  if (subtitle) {
    text(x + w / 2, y + 69, subtitle, { size: 12.5, weight: 600, fill: colors.muted })
  }
}

function arrow(d, options = {}) {
  const {
    dashed = false, color = colors.line, width = 2.4, marker = 'arrow', labelLines, labelX, labelY, labelAnchor = 'middle',
  } = options
  parts.push(`<path d="${d}" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round" ${dashed ? 'stroke-dasharray="8 7"' : ''} marker-end="url(#${marker})"/>`)
  if (labelLines && labelX !== undefined && labelY !== undefined) {
    label(labelX, labelY, labelLines, { anchor: labelAnchor })
  }
}

function panel(x, y, w, h, title, subtitle = '') {
  parts.push(`<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="24" fill="#FFFFFF" fill-opacity="0.66" stroke="#D8DEE8" stroke-width="1.5"/>`)
  text(x + 30, y + 42, title, { size: 20, weight: 800, anchor: 'start' })
  if (subtitle) {
    text(x + 30, y + 66, subtitle, { size: 12.5, weight: 550, fill: colors.muted, anchor: 'start' })
  }
}

function legend(x, y, fill, stroke, textValue) {
  parts.push(`<rect x="${x}" y="${y - 13}" width="18" height="18" rx="5" fill="${fill}" stroke="${stroke}" stroke-width="1.8"/>`)
  text(x + 27, y + 1, textValue, { size: 12.5, weight: 650, fill: colors.muted, anchor: 'start' })
}

parts.push(`<?xml version="1.0" encoding="UTF-8"?>`)
parts.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" role="img" aria-labelledby="title desc">`)
parts.push(`<title id="title">上门喂宠订单流转图</title><desc id="desc">任务、订单、支付、退款状态及其异常分支</desc>`)
parts.push(`<defs>
  <filter id="softShadow" x="-20%" y="-20%" width="140%" height="160%"><feDropShadow dx="0" dy="5" stdDeviation="7" flood-color="#0F172A" flood-opacity="0.10"/></filter>
  <marker id="arrow" markerWidth="10" markerHeight="10" refX="8.6" refY="5" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L10,5 L0,10 z" fill="${colors.line}"/></marker>
  <marker id="arrowRed" markerWidth="10" markerHeight="10" refX="8.6" refY="5" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L10,5 L0,10 z" fill="${colors.refundStroke}"/></marker>
  <style>text { font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Noto Sans CJK SC", "Microsoft YaHei", sans-serif; }</style>
</defs>`)
parts.push(`<rect width="${W}" height="${H}" fill="${colors.bg}"/>`)
parts.push(`<circle cx="1660" cy="80" r="180" fill="#FFF0D9" opacity="0.55"/><circle cx="1500" cy="1180" r="230" fill="#E5F7EE" opacity="0.48"/>`)

text(60, 58, '上门喂宠｜订单流转图', { size: 30, weight: 900, anchor: 'start', letter: 0.5 })
text(60, 88, '主链路、支付确认、超时抢占与退款补偿', { size: 14, weight: 600, fill: colors.muted, anchor: 'start' })

legend(1120, 58, colors.taskFill, colors.taskStroke, '任务状态')
legend(1250, 58, colors.paymentFill, colors.paymentStroke, '支付状态')
legend(1380, 58, colors.refundFill, colors.refundStroke, '退款状态')
legend(1510, 58, colors.neutralFill, colors.neutralStroke, '终态 / 关闭')

// Main task/order flow.
const mainY = 190
const mainH = 110
const main = [
  { x: 60, w: 230, title: 'BIDDING', sub: '竞价中 · 学生可报价', tag: '任务' },
  { x: 350, w: 270, title: 'PENDING_PAYMENT', sub: '订单 UNPAID · 已占学生时段', tag: '任务 + 订单' },
  { x: 680, w: 250, title: 'BOOKED', sub: '订单 PAID · 等待上门', tag: '任务 + 订单' },
  { x: 990, w: 220, title: 'IN_SERVICE', sub: '学生已签到', tag: '履约' },
  { x: 1270, w: 240, title: 'PENDING_CONFIRM', sub: '凭证已提交 · 待验收', tag: '履约' },
  { x: 1570, w: 180, title: 'COMPLETED', sub: '订单完成', tag: '终态', fill: colors.neutralFill, stroke: colors.neutralStroke },
]
for (const item of main) {
  stateBox(item.x, mainY, item.w, mainH, item.title, item.sub, {
    fill: item.fill || colors.taskFill,
    stroke: item.stroke || colors.taskStroke,
    tag: item.tag,
  })
}

const arrows = [
  { x1: 290, x2: 350, label: '选人锁定' },
  { x1: 620, x2: 680, label: '支付成功' },
  { x1: 930, x2: 990, label: '签到' },
  { x1: 1210, x2: 1270, label: '提交凭证' },
  { x1: 1510, x2: 1570, label: '验收' },
]
for (const item of arrows) {
  arrow(`M${item.x1} ${mainY + mainH / 2} H${item.x2 - 3}`, {
    labelLines: item.label, labelX: (item.x1 + item.x2) / 2, labelY: 172,
  })
}

// Main row explanatory strip.
parts.push(`<rect x="350" y="108" width="580" height="38" rx="14" fill="#FFF7ED" stroke="#FDBA74" stroke-width="1.4"/>`)
text(640, 133, '关键闸门：只有微信回调 / 主动查单确认金额与交易号后，UNPAID 才能进入 PAID → BOOKED', { size: 12.5, weight: 700, fill: '#9A3412' })

// Timeout and refund branches.
stateBox(250, 420, 250, 90, 'BIDDING', '超时未支付 · 距服务 > 1h', { tag: '释放时段', fill: colors.taskFill, stroke: colors.taskStroke })
stateBox(530, 420, 210, 90, 'EXPIRED', '超时未支付 · 距服务 ≤ 1h', { tag: '释放时段', fill: colors.neutralFill, stroke: colors.neutralStroke })
stateBox(800, 405, 340, 110, 'REFUNDING', '退款处理中 · 学生时段仍锁定', { tag: '退款', fill: colors.refundFill, stroke: colors.refundStroke, tagFill: colors.refundStroke })
stateBox(1250, 405, 230, 90, 'CANCELLED', '退款完成 · 任务终止', { tag: '退款成功', fill: colors.neutralFill, stroke: colors.neutralStroke })
stateBox(1250, 535, 230, 82, 'BOOKED', '普通退款失败 · 恢复预约', { tag: '仅限未开始', fill: colors.taskFill, stroke: colors.taskStroke })

arrow('M375 300 V417', { labelLines: '超时重开', labelX: 375, labelY: 392, color: colors.line })
arrow('M560 300 V380 H635 V417', { labelLines: '超时过期', labelX: 650, labelY: 398, labelAnchor: 'start', color: colors.line })
arrow('M600 300 V335 H850 V402', { labelLines: '截止后支付成功 → PAID_LATE', labelX: 725, labelY: 323, color: colors.refundStroke, marker: 'arrowRed' })
arrow('M900 300 V370 H1060 V402', { labelLines: '主人退款 / 审核开始', labelX: 930, labelY: 359, labelAnchor: 'start', color: colors.refundStroke, marker: 'arrowRed' })
arrow('M1140 460 H1247', { labelLines: '退款 SUCCESS', labelX: 1193, labelY: 447, color: colors.line })
arrow('M1080 515 V575 H1247', { labelLines: '退款 FAILED', labelX: 1160, labelY: 565, dashed: true, color: colors.line })

text(800, 543, '迟到支付只退旧订单，不影响任务新订单', { size: 12, weight: 600, fill: colors.muted, anchor: 'start' })

// Payment state machine.
panel(60, 640, 800, 510, '支付子状态', '微信支付 v3 · 固定商户单号 · 回调/查单幂等')
stateBox(100, 740, 170, 80, 'PREPARING', '创建本地支付单', { fill: colors.paymentFill, stroke: colors.paymentStroke, tag: '支付' })
stateBox(330, 740, 170, 80, 'PREPARED', '已取得 prepay_id', { fill: colors.paymentFill, stroke: colors.paymentStroke, tag: '支付' })
stateBox(570, 690, 220, 82, 'PAID', '订单 PAID · 任务 BOOKED', { fill: colors.paymentFill, stroke: colors.paymentStroke, tag: '截止前' })
stateBox(570, 835, 220, 82, 'PAID_LATE', '订单 REFUNDING · 自动退款', { fill: colors.refundFill, stroke: colors.refundStroke, tag: '截止后', tagFill: colors.refundStroke })
stateBox(330, 870, 170, 76, 'CLOSED', '超时关单', { fill: colors.neutralFill, stroke: colors.neutralStroke, tag: '终态' })
stateBox(100, 870, 170, 76, 'FAILED', '预下单异常', { fill: colors.neutralFill, stroke: colors.neutralStroke, tag: '可重试' })

arrow('M270 780 H327', { labelLines: '预下单', labelX: 299, labelY: 765 })
arrow('M500 760 H540 V731 H567', { labelLines: '成功', labelX: 532, labelY: 716 })
arrow('M500 800 H540 V876 H567', { labelLines: '迟到', labelX: 535, labelY: 852 })
arrow('M415 820 V867', { labelLines: '超时', labelX: 438, labelY: 847 })
arrow('M185 820 V867', { labelLines: '异常', labelX: 205, labelY: 847 })

text(100, 1005, '同一订单重复拉起收银台，始终复用同一个 out_trade_no', { size: 12.5, weight: 600, fill: colors.muted, anchor: 'start' })
text(100, 1034, '支付状态只信微信回调或服务端查单；客户端不能直接改 PAID', { size: 12.5, weight: 700, fill: '#166534', anchor: 'start' })
text(100, 1063, '迟到支付：PAID_LATE → 订单 REFUNDING → 自动生成 REQUESTED 退款', { size: 12.5, weight: 700, fill: '#9A3412', anchor: 'start' })

// Refund state machine.
panel(900, 640, 840, 510, '退款子状态', 'V1 人工审核 · 微信原路退款 · 处理中主动查单')
stateBox(940, 740, 180, 80, 'REQUESTED', '主人申请 / 自动生成', { fill: colors.refundFill, stroke: colors.refundStroke, tag: '待审核' })
stateBox(1200, 740, 180, 80, 'PROCESSING', '管理员已认领', { fill: colors.refundFill, stroke: colors.refundStroke, tag: '退款中' })
stateBox(1480, 690, 200, 82, 'SUCCESS', '订单退款 · 任务取消', { fill: colors.paymentFill, stroke: colors.paymentStroke, tag: '终态' })
stateBox(1480, 830, 200, 82, 'FAILED', '普通失败 · 恢复预约', { fill: colors.neutralFill, stroke: colors.neutralStroke, tag: '可重试' })
stateBox(940, 870, 180, 76, 'REJECTED', '管理员驳回', { fill: colors.neutralFill, stroke: colors.neutralStroke, tag: '终态' })

arrow('M1120 780 H1197', { labelLines: '审核通过', labelX: 1158, labelY: 765, color: colors.refundStroke, marker: 'arrowRed' })
arrow('M1380 760 H1425 V731 H1477', { labelLines: '退款成功', labelX: 1437, labelY: 716, color: colors.line })
arrow('M1380 800 H1425 V871 H1477', { labelLines: '异常 / 关闭', labelX: 1436, labelY: 848, color: colors.line })
arrow('M1480 871 C1400 930,1320 900,1290 823', { labelLines: '管理员重试', labelX: 1370, labelY: 925, dashed: true, color: colors.line })
arrow('M1030 820 V867', { labelLines: '驳回', labelX: 1055, labelY: 848, color: colors.line })

text(940, 1005, '迟到支付的退款不允许驳回，只能继续重试原路退款', { size: 12.5, weight: 700, fill: '#991B1B', anchor: 'start' })
text(940, 1034, 'out_refund_no 固定唯一；SUCCESS 重复回调直接返回原结果', { size: 12.5, weight: 600, fill: colors.muted, anchor: 'start' })
text(940, 1063, 'PROCESSING 每 30 秒查单，补偿丢失回调或进程重启', { size: 12.5, weight: 600, fill: colors.muted, anchor: 'start' })

// Final guarantees.
parts.push(`<rect x="60" y="1180" width="1680" height="42" rx="14" fill="#172033"/>`)
text(88, 1207, '最终防线', { size: 12.5, weight: 800, fill: '#FDE68A', anchor: 'start' })
text(170, 1207, '一个任务仅一个活动订单  ·  student_slots(student_id, slot_at) 唯一  ·  微信交易号/退款单号唯一  ·  重复回调幂等  ·  Redis 不是最终锁', { size: 12.5, weight: 650, fill: '#F8FAFC', anchor: 'start' })

parts.push('</svg>')

const output = path.resolve('docs/order-flow.svg')
fs.writeFileSync(output, parts.join('\n'))
console.log(output)
