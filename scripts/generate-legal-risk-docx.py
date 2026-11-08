from pathlib import Path

from docx import Document
from docx.enum.section import WD_ORIENT, WD_SECTION
from docx.enum.table import WD_CELL_VERTICAL_ALIGNMENT, WD_TABLE_ALIGNMENT
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from docx.shared import Cm, Pt, RGBColor


ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "docs" / "宠物喂养平台法律风险与责任矩阵.docx"

TEAL = "17362D"
GOLD = "C6A65A"
GOLD_SOFT = "F4ECD9"
PAPER = "F7F5F2"
WHITE = "FFFFFF"
INK = "25332F"
MUTED = "64716C"
RED = "A23A32"
RED_SOFT = "F8E8E5"
GREEN_SOFT = "EAF3EF"
LINE = "D9E1DD"
FONT = "Arial Unicode MS"


def color(value):
    return RGBColor.from_string(value)


def set_run(run, size=10, bold=False, text_color=INK, italic=False):
    run.font.name = "Arial"
    run.font.size = Pt(size)
    run.font.bold = bold
    run.font.italic = italic
    run.font.color.rgb = color(text_color)
    rpr = run._element.get_or_add_rPr().rFonts
    rpr.set(qn("w:eastAsia"), FONT)
    rpr.set(qn("w:ascii"), "Arial")
    rpr.set(qn("w:hAnsi"), "Arial")


def style_para(paragraph, before=0, after=5, line=1.2, keep=False):
    paragraph.paragraph_format.space_before = Pt(before)
    paragraph.paragraph_format.space_after = Pt(after)
    paragraph.paragraph_format.line_spacing = line
    paragraph.paragraph_format.keep_with_next = keep


def add_run(paragraph, text, size=10, bold=False, text_color=INK, italic=False):
    run = paragraph.add_run(text)
    set_run(run, size=size, bold=bold, text_color=text_color, italic=italic)
    return run


def shade(cell, fill):
    tc_pr = cell._tc.get_or_add_tcPr()
    shd = tc_pr.find(qn("w:shd"))
    if shd is None:
        shd = OxmlElement("w:shd")
        tc_pr.append(shd)
    shd.set(qn("w:fill"), fill)


def cell_margins(cell, top=90, start=105, bottom=90, end=105):
    tc_pr = cell._tc.get_or_add_tcPr()
    margin = tc_pr.first_child_found_in("w:tcMar")
    if margin is None:
        margin = OxmlElement("w:tcMar")
        tc_pr.append(margin)
    for name, value in (("top", top), ("start", start), ("bottom", bottom), ("end", end)):
        node = margin.find(qn(f"w:{name}"))
        if node is None:
            node = OxmlElement(f"w:{name}")
            margin.append(node)
        node.set(qn("w:w"), str(value))
        node.set(qn("w:type"), "dxa")


def cell_border(cell, border_color=LINE, size="4"):
    tc_pr = cell._tc.get_or_add_tcPr()
    borders = tc_pr.first_child_found_in("w:tcBorders")
    if borders is None:
        borders = OxmlElement("w:tcBorders")
        tc_pr.append(borders)
    for edge in ("top", "left", "bottom", "right", "insideH", "insideV"):
        node = borders.find(qn(f"w:{edge}"))
        if node is None:
            node = OxmlElement(f"w:{edge}")
            borders.append(node)
        node.set(qn("w:val"), "single")
        node.set(qn("w:sz"), size)
        node.set(qn("w:space"), "0")
        node.set(qn("w:color"), border_color)


def set_width(cell, width):
    cell.width = width
    tc_pr = cell._tc.get_or_add_tcPr()
    tc_w = tc_pr.find(qn("w:tcW"))
    if tc_w is None:
        tc_w = OxmlElement("w:tcW")
        tc_pr.append(tc_w)
    tc_w.set(qn("w:w"), str(int(width.twips)))
    tc_w.set(qn("w:type"), "dxa")


def repeat_header(row):
    tr_pr = row._tr.get_or_add_trPr()
    node = OxmlElement("w:tblHeader")
    node.set(qn("w:val"), "true")
    tr_pr.append(node)


def no_split(row):
    tr_pr = row._tr.get_or_add_trPr()
    tr_pr.append(OxmlElement("w:cantSplit"))


def write_cell(cell, value, size=8.2, bold=False, text_color=INK):
    lines = value if isinstance(value, (list, tuple)) else [value]
    cell.text = ""
    paragraph = cell.paragraphs[0]
    style_para(paragraph, after=1.4, line=1.06)
    for index, line in enumerate(lines):
        if index:
            paragraph = cell.add_paragraph()
            style_para(paragraph, after=1.4, line=1.06)
        add_run(paragraph, line, size=size, bold=bold, text_color=text_color)
    cell.vertical_alignment = WD_CELL_VERTICAL_ALIGNMENT.CENTER


def add_table(document, headers, rows, widths, size=8.2, first_bold=False):
    table = document.add_table(rows=1, cols=len(headers))
    table.style = "Table Grid"
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    table.autofit = False
    header_row = table.rows[0]
    repeat_header(header_row)
    for index, header in enumerate(headers):
        cell = header_row.cells[index]
        set_width(cell, widths[index])
        cell_margins(cell, top=115, bottom=115)
        cell_border(cell, border_color=TEAL)
        shade(cell, TEAL)
        cell.text = ""
        paragraph = cell.paragraphs[0]
        paragraph.alignment = WD_ALIGN_PARAGRAPH.CENTER
        style_para(paragraph, after=0, line=1.0)
        add_run(paragraph, header, size=size + 0.2, bold=True, text_color=WHITE)
    for row_index, values in enumerate(rows):
        row = table.add_row()
        no_split(row)
        for col_index, value in enumerate(values):
            cell = row.cells[col_index]
            set_width(cell, widths[col_index])
            cell_margins(cell)
            cell_border(cell, border_color=LINE, size="3")
            shade(cell, WHITE if row_index % 2 == 0 else PAPER)
            write_cell(cell, value, size=size, bold=(first_bold and col_index == 0))
    return table


def heading(document, number, title, kicker):
    paragraph = document.add_paragraph()
    style_para(paragraph, after=1, keep=True)
    add_run(paragraph, kicker, size=7.8, bold=True, text_color=GOLD)
    paragraph = document.add_paragraph()
    style_para(paragraph, after=7, keep=True)
    add_run(paragraph, f"{number}  ", size=18, bold=True, text_color=GOLD)
    add_run(paragraph, title, size=18, bold=True, text_color=TEAL)
    p_pr = paragraph._p.get_or_add_pPr()
    p_bdr = OxmlElement("w:pBdr")
    bottom = OxmlElement("w:bottom")
    bottom.set(qn("w:val"), "single")
    bottom.set(qn("w:sz"), "8")
    bottom.set(qn("w:space"), "5")
    bottom.set(qn("w:color"), LINE)
    p_bdr.append(bottom)
    p_pr.append(p_bdr)


def callout(document, title, lines, fill=GOLD_SOFT):
    table = document.add_table(rows=1, cols=1)
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    table.autofit = False
    cell = table.cell(0, 0)
    set_width(cell, Cm(17.4))
    shade(cell, fill)
    cell_border(cell, border_color="E6D7B5")
    cell_margins(cell, top=170, bottom=160, start=210, end=210)
    cell.text = ""
    paragraph = cell.paragraphs[0]
    style_para(paragraph, after=5, line=1.18)
    add_run(paragraph, title, size=11, bold=True, text_color=TEAL)
    for line in lines:
        paragraph = cell.add_paragraph()
        style_para(paragraph, after=2, line=1.18)
        add_run(paragraph, line, size=9.3)


def bullet(document, text, size=9.6):
    paragraph = document.add_paragraph(style="List Bullet")
    style_para(paragraph, after=3, line=1.18)
    add_run(paragraph, text, size=size)


def add_page_field(paragraph):
    run = paragraph.add_run()
    begin = OxmlElement("w:fldChar")
    begin.set(qn("w:fldCharType"), "begin")
    instruction = OxmlElement("w:instrText")
    instruction.set(qn("xml:space"), "preserve")
    instruction.text = "PAGE"
    separate = OxmlElement("w:fldChar")
    separate.set(qn("w:fldCharType"), "separate")
    text = OxmlElement("w:t")
    text.text = "1"
    end = OxmlElement("w:fldChar")
    end.set(qn("w:fldCharType"), "end")
    for node in (begin, instruction, separate, text, end):
        run._r.append(node)
    set_run(run, size=8, text_color=MUTED)


def header_footer(section):
    section.header.is_linked_to_previous = False
    section.footer.is_linked_to_previous = False
    paragraph = section.header.paragraphs[0]
    paragraph.text = ""
    style_para(paragraph, after=0, line=1.0)
    add_run(paragraph, "上门喂宠平台  /  法律风险、连带责任与防控清单", size=7.8, text_color=MUTED)
    p_pr = paragraph._p.get_or_add_pPr()
    p_bdr = OxmlElement("w:pBdr")
    bottom = OxmlElement("w:bottom")
    bottom.set(qn("w:val"), "single")
    bottom.set(qn("w:sz"), "4")
    bottom.set(qn("w:space"), "4")
    bottom.set(qn("w:color"), LINE)
    p_bdr.append(bottom)
    p_pr.append(p_bdr)
    paragraph = section.footer.paragraphs[0]
    paragraph.text = ""
    paragraph.alignment = WD_ALIGN_PARAGRAPH.CENTER
    style_para(paragraph, after=0, line=1.0)
    add_run(paragraph, "内部讨论稿 · 第 ", size=8, text_color=MUTED)
    add_page_field(paragraph)
    add_run(paragraph, " 页 · 不构成正式法律意见", size=8, text_color=MUTED)


def page_number_start(section, number=1):
    pg_num = section._sectPr.find(qn("w:pgNumType"))
    if pg_num is None:
        pg_num = OxmlElement("w:pgNumType")
        section._sectPr.append(pg_num)
    pg_num.set(qn("w:start"), str(number))


def continue_page_numbering(section):
    pg_num = section._sectPr.find(qn("w:pgNumType"))
    if pg_num is not None:
        section._sectPr.remove(pg_num)


def configure_a4(section, landscape=False):
    section.orientation = WD_ORIENT.LANDSCAPE if landscape else WD_ORIENT.PORTRAIT
    section.page_width = Cm(29.7 if landscape else 21.0)
    section.page_height = Cm(21.0 if landscape else 29.7)
    section.top_margin = Cm(1.45 if landscape else 1.8)
    section.bottom_margin = Cm(1.3 if landscape else 1.65)
    section.left_margin = Cm(1.35 if landscape else 1.8)
    section.right_margin = Cm(1.35 if landscape else 1.8)


document = Document()
document.core_properties.title = "上门喂宠平台法律风险、连带责任与防控清单"
document.core_properties.subject = "平台法律风险、连带责任、风险矩阵与整改优先级"
document.core_properties.author = "平台运营与产品团队"
document.core_properties.keywords = "宠物喂养, 平台责任, 连带责任, 个人信息保护, 风险矩阵"
document.core_properties.comments = "内部讨论稿，不构成正式法律意见。"

normal = document.styles["Normal"]
normal.font.name = "Arial"
normal.font.size = Pt(10)
normal._element.rPr.rFonts.set(qn("w:eastAsia"), FONT)
normal.paragraph_format.space_after = Pt(5)
normal.paragraph_format.line_spacing = 1.2

# Cover
cover = document.sections[0]
configure_a4(cover)
cover.top_margin = Cm(2.2)
cover.left_margin = Cm(2.2)
cover.right_margin = Cm(2.2)

paragraph = document.add_paragraph()
style_para(paragraph, before=18, after=24)
add_run(paragraph, "ZOO PLATFORM  /  INTERNAL RISK REVIEW", size=9, bold=True, text_color=GOLD)

paragraph = document.add_paragraph()
style_para(paragraph, before=0, after=3)
add_run(paragraph, "法律风险、连带责任", size=29, bold=True, text_color=TEAL)
paragraph = document.add_paragraph()
style_para(paragraph, before=0, after=15)
add_run(paragraph, "与平台防控清单", size=29, bold=True, text_color=GOLD)
paragraph = document.add_paragraph()
style_para(paragraph, before=0, after=22)
add_run(paragraph, "面向“主人在线发单 + 学生上门服务 + 平台竞价与支付”的业务模式", size=11, text_color=MUTED)

paragraph = document.add_paragraph()
style_para(paragraph, before=0, after=28)
p_pr = paragraph._p.get_or_add_pPr()
p_bdr = OxmlElement("w:pBdr")
bottom = OxmlElement("w:bottom")
bottom.set(qn("w:val"), "single")
bottom.set(qn("w:sz"), "16")
bottom.set(qn("w:space"), "1")
bottom.set(qn("w:color"), GOLD)
p_bdr.append(bottom)
p_pr.append(p_bdr)

meta_rows = [
    ("文件属性", "内部讨论稿", "版本", "V1.0"),
    ("适用法域", "中国大陆", "日期", "2026-11-05"),
    ("适用业务", "宠物上门照护撮合", "复核状态", "待律师复核"),
]
meta = document.add_table(rows=3, cols=4)
meta.autofit = False
for row_index, values in enumerate(meta_rows):
    for col_index, value in enumerate(values):
        cell = meta.cell(row_index, col_index)
        set_width(cell, Cm(4.2))
        cell_margins(cell, top=100, bottom=100, start=120, end=100)
        cell_border(cell, border_color=LINE, size="3")
        shade(cell, PAPER)
        cell.text = ""
        paragraph = cell.paragraphs[0]
        style_para(paragraph, after=0, line=1.0)
        add_run(paragraph, value, size=8.8, bold=(col_index % 2 == 0), text_color=MUTED if col_index % 2 == 0 else TEAL)

paragraph = document.add_paragraph()
style_para(paragraph, before=34, after=4)
add_run(paragraph, "重要提示", size=9, bold=True, text_color=RED)
paragraph = document.add_paragraph()
style_para(paragraph, before=0, after=0, line=1.16)
add_run(
    paragraph,
    "本文件是基于当前产品模式的风险识别与合规准备清单，不构成正式法律意见。平台、学生、主人之间的实际责任取决于运营主体、合同关系、收费模式、平台控制程度、证据和个案事实，上线前应由专业律师结合业务做专项复核。",
    size=8.2,
    text_color=MUTED,
)

# Main portrait section
main = document.add_section(WD_SECTION.NEW_PAGE)
configure_a4(main)
page_number_start(main, 1)
header_footer(main)

heading(document, "01", "执行摘要", "EXECUTIVE SUMMARY")
callout(
    document,
    "一句话结论",
    [
        "本平台最危险的不是 KM 派单算法，而是“让陌生人进入宠物主人家 + 平台掌握精确地址与学生身份 + 平台经手交易资金”。",
        "算法失误通常只造成匹配效果下降；入室安全、宠物伤亡、个人信息泄露和资金结算问题，可能直接带来民事赔偿、行政处罚、刑事责任和公关危机。",
    ],
)

paragraph = document.add_paragraph()
style_para(paragraph, before=9, after=3, keep=True)
add_run(paragraph, "核心判断", size=12.5, bold=True, text_color=TEAL)
for text in [
    "平台收佣金本身不等于当然承担连带责任。真正高风险的是：平台知道或应当知道风险，却不采取必要措施。",
    "平台控制越深，责任越重。若平台决定价格、强制派单、统一培训、统一收款并限制学生拒单，可能被认定为共同经营者或实际用工方，而不只是信息中介。",
    "免责条款不能排除法定责任。人身损害、隐私泄露、消费欺诈等责任，不能仅靠用户协议中的“平台不担责”完全排除。",
    "当前 5km 过滤、成交前隐藏门牌、支付状态机和 KM 只读预演是有效控制；但管理员单令牌、敏感数据明文、人工学籍审核和 UGC 无审核仍需要补齐。",
]:
    bullet(document, text)

paragraph = document.add_paragraph()
style_para(paragraph, before=9, after=4, keep=True)
add_run(paragraph, "当前产品控制与主要缺口", size=12.5, bold=True, text_color=TEAL)
add_table(
    document,
    ["已有控制", "降低的风险", "仍需补齐"],
    [
        ["成交前隐藏详细门牌", "降低地址被提前收集、尾随和骚扰风险", "临时门牌可见、离场后回收、异常访问告警"],
        ["服务端 5km 距离校验", "降低跨区接单和绕过前端风险", "定位时效、GPS 防伪、道路距离、跨河/高速场景"],
        ["唯一订单与支付状态机", "降低重复付款、重复接单和资金错账", "每日对账、学生合规结算、税务与发票方案"],
        ["KM 只读派单对比", "降低算法直接造成履约事故的风险", "若改为自动派单，需增加复核、超时释放和通知兜底"],
        ["学籍材料人工审核", "降低无学籍学生进入平台的概率", "实名/活体、本人一致性、18 岁校验、周期复核"],
        ["静态 ADMIN_TOKEN 后台", "开发阶段便于管理", "RBAC、MFA、操作人审计、双人复核、离职回收"],
    ],
    [Cm(4.6), Cm(5.1), Cm(7.3)],
    size=7.9,
)

document.add_page_break()
heading(document, "02", "责任类型与判断顺序", "LIABILITY MODEL")
paragraph = document.add_paragraph()
style_para(paragraph, after=6)
add_run(
    paragraph,
    "《民法典》第 178 条规定，连带责任必须有法律规定或当事人约定，不能因为平台收费、撮合或使用自己的品牌就自动推定。实践中先判断平台的角色，再判断平台是否有过错以及过错与损害之间是否存在因果关系。",
    size=9.5,
)
add_table(
    document,
    ["责任类型", "受害人可以怎么主张", "平台什么时候容易触发", "典型场景"],
    [
        ["直接责任", "平台自己的行为或合同义务造成损害，受害人直接向平台主张", "泄露数据、错误扣款、虚假宣传、平台员工侵权、平台自营或明确承诺结果", "隐私泄露、退款错误、平台自营服务事故"],
        ["连带责任", "可以向平台或实际服务者任何一方要求全部赔偿；平台后可追偿", "知道或应当知道风险仍不处理；与第三方共同决定个人信息处理目的和方式；法律明确规定连带", "明知学生有暴力/盗窃投诉仍继续匹配、共同处理个人信息、共同发布侵权内容"],
        ["相应 / 补充责任", "先由实际服务者赔付；不足部分在平台过错和原因力范围内承担，赔完可追偿", "未核验身份或资质、未履行安全保障义务；具体是“相应责任”还是“补充责任”需结合法条与个案", "假学籍、冒名学生造成损失；审核和投诉处置明显失职"],
        ["先行赔付", "消费者可先要求平台赔付，平台再向实际经营者追偿", "不能提供平台内经营者的真实名称、地址和有效联系方式", "主人无法找到实际服务者，平台也无法提供真实身份和联系信息"],
        ["行政 / 刑事责任", "监管罚款、整改、暂停业务、责任人处罚；严重时进入刑事程序", "个人信息违法处理、无证支付、虚假广告、明知犯罪仍提供帮助", "数据黑产、二清、协助诈骗、传播违法内容"],
    ],
    [Cm(3.1), Cm(5.1), Cm(5.7), Cm(3.1)],
    size=7.9,
    first_bold=True,
)

paragraph = document.add_paragraph()
style_para(paragraph, before=8, after=3, keep=True)
add_run(paragraph, "建议的判断顺序", size=12, bold=True, text_color=TEAL)
for index, text in enumerate([
    "平台是单纯信息撮合，还是实际参与服务、定价、派单、收款和人员管理？",
    "平台是否知道或应当知道某个学生、任务或内容存在明显风险？",
    "平台收到投诉、举报或异常信号后，是否及时采取了必要措施？",
    "平台是否履行了身份、学籍、年龄、服务资质和安全能力核验？",
    "平台与第三方是否共同决定个人信息处理目的和方式？",
], 1):
    paragraph = document.add_paragraph()
    style_para(paragraph, after=2, line=1.15)
    add_run(paragraph, f"{index}. ", size=9.5, bold=True, text_color=GOLD)
    add_run(paragraph, text, size=9.4)

document.add_page_break()
heading(document, "03", "主要法律依据（节选）", "LEGAL BASIS")
paragraph = document.add_paragraph()
style_para(paragraph, after=6)
add_run(paragraph, "以下条款用于风险识别，不是对个案责任的最终判断；具体适用、责任比例和抗辩空间需由律师结合合同、证据与监管口径复核。", size=8.9, text_color=MUTED)
add_table(
    document,
    ["规范", "核心条款", "与平台的关系", "对当前业务的意义"],
    [
        ["《民法典》", "第 178、1191、1195、1197、1245 条", "连带责任法定/约定、用工责任、网络侵权通知规则、动物致害责任", "处理学生致害、宠物伤人、社区侵权、员工/外包侵权的基础框架"],
        ["《电子商务法》", "第 27、38 条", "身份资质审核、知情不处置的连带责任及安全保障责任", "学生身份、学籍、服务能力的持续审核，以及投诉后的必要处置"],
        ["《消费者权益保护法》", "第 44 条", "不能提供实际经营者真实信息时的先行赔付；明知侵权不处理的连带责任", "主人找不到实际服务者时，平台可能先承担赔付压力"],
        ["《个人信息保护法》", "第 20、21、28、29、51、55、57、69 条", "共同处理连带、委托处理、敏感信息、单独同意、安全措施、影响评估、泄露通知、过错推定", "地址、定位、学号、证件、人脸和支付信息都应按敏感数据管理"],
        ["《网络交易监督管理办法》", "平台责任相关条款", "平台治理、信息公示、交易记录、消费者权益保护", "持续撮合交易可能受到平台义务约束"],
        ["《广告法》《反不正当竞争法》", "虚假宣传、误导性展示相关条款", "“全程保障”“专业认证”“绝对安全”等宣传必须可验证", "避免把平台保障描述成无法兑现的承诺"],
        ["动物防疫、执业兽医相关规范", "动物诊疗与用药要求", "生活照护与医疗行为边界需要明确", "禁止诊断、注射、处方药等超出生活照护的行为"],
        ["税收征管、劳动与社保规范", "扣缴义务、劳动关系认定相关规则", "平台对学生的结算模式可能影响用工与税务责任", "避免名为合作、实为固定排班和强控制"],
    ],
    [Cm(3.0), Cm(4.3), Cm(5.5), Cm(4.2)],
    size=7.8,
)

# Landscape risk matrix
landscape = document.add_section(WD_SECTION.NEW_PAGE)
configure_a4(landscape, landscape=True)
continue_page_numbering(landscape)
header_footer(landscape)
heading(document, "04", "法律风险与连带责任矩阵", "RISK MATRIX")
paragraph = document.add_paragraph()
style_para(paragraph, after=4)
add_run(paragraph, "P0 表示上线前应优先处理；P1 表示可与运营同步建设。“连带”与“相应责任”均需以具体法条、合同和证据为前提。", size=8.5, text_color=MUTED)

risks = [
    ["P0", ["学生盗窃、骚扰、故意伤害", "上门服务中的入室安全事件"], "学生个人", "学生直接承担刑事/民事责任；平台知情或应知仍继续派单可能连带；实际招用、强管理形成用工关系时，平台承担用工责任。", "投诉后继续派单；未做身份/活体核验；平台统一培训、定价、禁止拒单", "实名+活体、投诉筛查、培训考试、拉黑、订单熔断、SOS、签到与离场留证、保险"],
    ["P0", ["宠物咬伤/抓伤学生", "或学生上门时受伤"], "宠物饲养人/管理人；有过错的业主", "通常不当然连带；平台未提示危险宠物、仍推荐或派单，可能承担相应责任；合同或宣传形成安全保障承诺时风险更高。", "主人隐瞒攻击史；平台仍推荐“安全服务”；未提供防护规范；未提示急救", "宠物行为问卷、烈性犬与危险服务限制、服务前确认、防护装备、意外险、紧急联系人"],
    ["P0", ["喂养错误、忘喂、用药不当", "宠物走失或死亡"], "实际服务学生；有错误指示的主人", "学生/主人按过错承担；平台知情不处置可能连带，审核或安全保障失职可能承担相应责任；平台自营或承诺结果时可能直接负责。", "无服务清单；无到场/离场记录；无照片凭证；平台宣称“全程保障”", "标准 SOP、清单确认、禁止医疗行为、签到签退、照片/视频、异常上报、兽医授权和费用上限"],
    ["P0", ["假学生、冒名学生", "未成年人接单"], "冒名者个人", "平台负身份与资质审核义务；未尽审核可能承担相应责任；明知仍提供服务可能连带；构成欺诈可能触发惩罚性赔偿。", "只验证学信网验证码；不验证本人和年龄；认证过期不复查", "手机号+实名/活体+18 岁校验、学籍周期复核、异常设备识别、人工抽检"],
    ["P0", ["精确地址、定位、学号", "身份材料、照片泄露"], "平台及实际泄露人员", "平台处理个人信息通常直接负责；与第三方共同决定处理目的和方式，可能依《个人信息保护法》第 20 条承担连带责任。", "明文存储；后台权限过大；员工私查；上传文件公开；外包无协议", "敏感字段加密、最小权限、MFA、访问审计、外包 DPA、泄露通知与应急预案"],
    ["P0", ["社区造谣、辱骂", "泄露地址、侵权图片"], "发布用户", "用户直接负责；平台收到有效通知后不删除，可能对扩大损害连带；明知/应知仍推荐可能连带；主动编辑推荐时可能直接成为内容发布者。", "无举报入口；投诉不处理；无审核；平台加热侵权内容", "内容审核、举报下架、重复侵权处置、社区规则、证据留存、申诉机制"],
    ["P0", ["支付、退款、提现", "抽佣、二清"], "平台运营主体", "多数为平台直接合同/监管责任，不是当然连带；无证支付、二清、资金池可能触发行政或刑事风险；明知协助欺诈可能进一步扩大责任。", "个人账户收款；平台代收代付不清；退款无账本；学生提现无合规结算", "微信支付合规模式、公司账户、订单账本、每日对账、退款双人复核、财税方案"],
    ["P1", ["平台与学生被认定构成劳动关系", "工资、工伤、社保、个税"], "实际用工主体", "若构成劳动关系，平台直接承担工资、工伤和社保风险；劳务派遣/外包还会涉及派遣单位和用工单位责任。", "强制排班；禁止拒单；统一工服；固定工资；强考核；名为合作实为用工", "明确合作边界、保留自主接单、按单结算、劳动法律师审查合同和实际流程"],
    ["P1", ["无证提供宠物医疗", "打针、用药"], "实际服务者；平台若组织或宣传", "实际服务者承担主要行政/民事责任；平台明知仍组织、宣传或撮合，可能承担连带或相应责任。", "服务项目包含诊断、注射、处方药；平台宣传“专业医疗照护”", "明确禁止医疗行为；只做喂食、换水、清洁；紧急情况送医；保留主人授权"],
    ["P1", ["虚假宣传、算法派单不公平", "差别定价"], "平台运营主体", "主要是平台直接行政和民事责任；平台控制越强、宣传越绝对，注意义务越高；自动派单错误可能扩大履约风险。", "“全部实名认证”“平台担保安全”但实际未做到；算法长期排斥新学生", "宣传可验证、算法留痕、人工复核、申诉机制、公平性监控"],
    ["P1", ["管理员或外包人员私查", "泄露、盗用数据"], "个人及平台", "个人承担侵权/刑事责任；平台承担管理、合同和用工责任，并可向个人追偿；构成共同侵权时可能连带。", "单管理员令牌；无操作人记录；离职不回收；外包无审计", "RBAC、MFA、实名审计、双人审批、离职回收、日志防篡改、异常导出告警"],
]
add_table(
    document,
    ["等级", "风险场景", "主要责任主体", "平台可能责任性质", "高风险触发条件", "最低防控动作"],
    risks,
    [Cm(1.2), Cm(4.5), Cm(2.8), Cm(6.7), Cm(5.4), Cm(5.5)],
    size=7.25,
    first_bold=True,
)

# Final portrait section
final = document.add_section(WD_SECTION.NEW_PAGE)
configure_a4(final)
continue_page_numbering(final)
header_footer(final)
heading(document, "05", "关键结论与整改优先级", "ACTION PLAN")
callout(
    document,
    "四条必须写进管理层共识的结论",
    [
        "1. 平台抽佣不等于当然连带；但平台知情、应知、强控制或共同处理数据时，责任会明显上升。",
        "2. “平台只提供信息”不能仅靠一句免责声明成立，必须与合同、收费、派单、培训、结算和实际管理方式保持一致。",
        "3. 对关系消费者生命健康的服务，平台的身份/资质审核和安全保障义务是硬义务；宠物上门照护属于高风险场景。",
        "4. 当前最优先补的是实名+年龄、管理员安全、地址与定位加密、投诉处置、保险和资金结算合规，而不是继续增加算法复杂度。",
    ],
    fill=GREEN_SOFT,
)
paragraph = document.add_paragraph()
style_para(paragraph, before=8, after=3, keep=True)
add_run(paragraph, "分阶段落地清单", size=12, bold=True, text_color=TEAL)
add_table(
    document,
    ["阶段", "建议动作", "验收标准"],
    [
        ["0–7 天\n止血", "立即重置已暴露的微信 AppSecret；更换强随机 ADMIN_TOKEN；关闭 DEV_AUTH；检查 .env.local 权限；建立事故联系人和证据保存目录", "生产环境无默认密钥；密钥只在受控环境变量；每日备份可恢复"],
        ["0–30 天\n准入与合同", "实名/活体/18 岁校验；学生服务协议；主人服务协议；隐私政策；宠物健康与攻击性问卷；保险询价；律师复核", "没有实名和年龄信息不能接单；关键页面完成授权与同意留痕"],
        ["0–30 天\n后台与数据", "管理员账号化、RBAC、MFA、会话过期、操作人/IP/请求 ID 审计；地址、证件、定位加密；最小化导出", "敏感数据访问可追溯到具体管理员；离职 24 小时内回收权限"],
        ["30–60 天\n运营安全", "投诉/举报/下架/拉黑/熔断；高风险订单人工复核；紧急送医流程；夜间和危险服务限制；服务照片与签到签退", "投诉闭环有 SLA；每起事故有证据链、责任人和复盘报告"],
        ["30–90 天\n资金与稳定性", "微信支付合规结算；每日对账；退款双人复核；学生税务/发票方案；PostgreSQL、对象存储、监控告警", "账实一致；支付/退款异常自动告警；单机故障可在目标时间内恢复"],
    ],
    [Cm(2.8), Cm(10.2), Cm(4.1)],
    size=7.9,
    first_bold=True,
)

heading(document, "06", "事故与投诉处置底线", "INCIDENT RESPONSE")
paragraph = document.add_paragraph()
style_para(paragraph, after=5)
add_run(paragraph, "发生宠物伤亡、人员伤害、入室盗窃、严重隐私泄露或群体投诉时，先保护人和动物，再谈责任划分。", size=9.5, bold=True, text_color=RED)
for title, detail in [
    ("先救人救宠物", "立即联系主人、学生、紧急联系人和合作宠物医院；必要时报警、叫救护车；保留现场，不诱导改口或销毁证据。"),
    ("暂停账号与派单", "冻结争议订单、暂停涉事学生接单、保留聊天和操作日志，避免继续扩大损害。"),
    ("固定证据", "保存任务记录、签到/离场、照片、支付与退款流水、投诉时间线、客服沟通和管理员操作日志。"),
    ("同步保险与律师", "在保险报案时限内通知保险人；涉及人身伤害、死亡、刑事线索或重大舆情时，让律师第一时间介入。"),
    ("对外沟通克制", "事实未查清前不要承诺“平台全责”或公开学生/主人隐私，统一由指定发言人沟通。"),
    ("复盘并修复根因", "区分单次事件与系统性问题，更新准入、SOP、算法、权限、保险和合同，不只做口头警告。"),
]:
    paragraph = document.add_paragraph()
    style_para(paragraph, after=3, line=1.16)
    add_run(paragraph, f"• {title}：", size=9.4, bold=True, text_color=TEAL)
    add_run(paragraph, detail, size=9.2)

heading(document, "07", "上线前复核问题", "SIGN-OFF")
for text in [
    "平台运营主体、收款主体、开票主体和保险投保主体是否一致？",
    "主人、学生、平台各自签的是服务合同、居间合同还是劳动合同？实际执行是否与合同一致？",
    "平台是否能提供实际服务者的真实姓名、有效联系方式和身份信息？",
    "学生是否完成实名、年龄、学籍有效期和本人一致性校验？",
    "地址、定位、证件和学信网材料是否完成最小化收集、加密、权限与保留期限设计？",
    "社区内容是否有审核、举报、下架、申诉和重复侵权处置机制？",
    "支付、退款、提现、抽佣是否有合规结算、对账和税务方案？",
    "平台是否购买与业务匹配的责任险、意外险，并核清免赔和除外责任？",
    "发生宠物死亡、人员伤害或数据泄露时，谁负责报警、送医、通知、公关和证据保全？",
]:
    paragraph = document.add_paragraph()
    style_para(paragraph, after=3, line=1.14)
    add_run(paragraph, "□  ", size=10, bold=True, text_color=GOLD)
    add_run(paragraph, text, size=9.3)

paragraph = document.add_paragraph()
style_para(paragraph, before=12, after=3)
add_run(paragraph, "复核签署", size=10.5, bold=True, text_color=TEAL)
sign = document.add_table(rows=1, cols=4)
sign.autofit = False
for index, (label, width) in enumerate([
    ("运营负责人：", Cm(4.5)),
    ("技术负责人：", Cm(4.5)),
    ("法律顾问：", Cm(4.5)),
    ("日期：", Cm(3.8)),
]):
    cell = sign.cell(0, index)
    set_width(cell, width)
    cell_margins(cell, top=160, bottom=160, start=140, end=140)
    cell_border(cell, border_color=LINE, size="3")
    shade(cell, PAPER)
    cell.text = ""
    paragraph = cell.paragraphs[0]
    style_para(paragraph, after=0, line=1.0)
    add_run(paragraph, label, size=8.3, text_color=MUTED)

paragraph = document.add_paragraph()
style_para(paragraph, before=16, after=0, line=1.15)
add_run(
    paragraph,
    "附录说明：本文基于当前产品模式做概括性风险分析，引用法规为风险识别所需的条款节选，未穷尽全部法律法规、司法解释和地方监管要求。正式上线前请由执业律师、税务顾问和保险经纪人进行专项复核。",
    size=7.8,
    text_color=MUTED,
)

settings = document.settings._element
update_fields = settings.find(qn("w:updateFields"))
if update_fields is None:
    update_fields = OxmlElement("w:updateFields")
    settings.append(update_fields)
update_fields.set(qn("w:val"), "true")

OUT.parent.mkdir(parents=True, exist_ok=True)
document.save(OUT)
print(OUT)
