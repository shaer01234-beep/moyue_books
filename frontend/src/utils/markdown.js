// 轻量 Markdown 渲染器（无第三方依赖）
// 支持：标题、粗体、斜体、删除线、行内代码、代码块、引用、无序/有序列表、链接、图片、分割线、简单表格
// 安全策略：先对原始内容做 HTML 转义，再生成标签，杜绝 XSS 注入

function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// 行内解析：在已转义文本上应用粗体/斜体/删除线/行内代码/链接/图片
function renderInline(text) {
  let t = text
  // 行内代码（优先，避免内部符号被误解析）
  t = t.replace(/`([^`]+)`/g, (m, code) => `<code>${code}</code>`)
  // 图片 ![alt](url)
  t = t.replace(/!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g, (m, alt, url, title) => {
    const safe = escapeHtml(url)
    const tip = title ? ` title="${escapeHtml(title)}"` : ''
    return `<img src="${safe}" alt="${escapeHtml(alt)}"${tip} loading="lazy">`
  })
  // 链接 [text](url)
  t = t.replace(/\[([^\]]+)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g, (m, text, url, title) => {
    const safe = escapeHtml(url)
    const tip = title ? ` title="${escapeHtml(title)}"` : ''
    return `<a href="${safe}" target="_blank" rel="noopener noreferrer"${tip}>${text}</a>`
  })
  // 粗体 **text**
  t = t.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  // 斜体 *text* 或 _text_
  t = t.replace(/(^|[^*])\*([^*\n]+)\*(?!\*)/g, '$1<em>$2</em>')
  t = t.replace(/(^|[^_])_([^_\n]+)_(?!_)/g, '$1<em>$2</em>')
  // 删除线 ~~text~~
  t = t.replace(/~~([^~]+)~~/g, '<del>$1</del>')
  return t
}

// 块级解析：返回 HTML
export function renderMarkdown(md) {
  if (!md) return ''
  const raw = String(md).replace(/\r\n/g, '\n')

  const lines = raw.split('\n')
  const html = []
  let i = 0
  let listType = '' // '', 'ul', 'ol'
  let inCode = false
  let codeBuf = []
  let inQuote = false
  let quoteBuf = []
  let inTable = false
  let tableBuf = []
  let paragraph = []

  const closeList = () => {
    if (listType) {
      html.push(`</${listType}>`)
      listType = ''
    }
  }
  const closeQuote = () => {
    if (inQuote) {
      html.push(`<blockquote>${quoteBuf.join('\n')}</blockquote>`)
      quoteBuf = []
      inQuote = false
    }
  }
  const closeTable = () => {
    if (inTable) {
      if (tableBuf.length >= 2) {
        const header = tableBuf[0]
        const body = tableBuf.slice(1)
        const colCount = header.length
        html.push('<table><thead><tr>')
        for (const cell of header) html.push(`<th>${renderInline(cell.trim())}</th>`)
        html.push('</tr></thead><tbody>')
        for (const row of body) {
          html.push('<tr>')
          for (let ci = 0; ci < colCount; ci++) {
            html.push(`<td>${renderInline((row[ci] || '').trim())}</td>`)
          }
          html.push('</tr>')
        }
        html.push('</tbody></table>')
      }
      tableBuf = []
      inTable = false
    }
  }
  const flushParagraph = () => {
    if (paragraph.length) {
      html.push(`<p>${renderInline(paragraph.join(' '))}</p>`)
      paragraph = []
    }
  }

  while (i < lines.length) {
    const line = lines[i]

    // 代码块
    if (/^```/.test(line)) {
      if (inCode) {
        html.push(`<pre><code>${escapeHtml(codeBuf.join('\n'))}</code></pre>`)
        codeBuf = []
        inCode = false
      } else {
        flushParagraph()
        closeList()
        closeQuote()
        closeTable()
        inCode = true
      }
      i++
      continue
    }
    if (inCode) {
      codeBuf.push(line)
      i++
      continue
    }

    const trimmed = line.trim()

    // 空行：结束当前块
    if (trimmed === '') {
      flushParagraph()
      closeList()
      closeQuote()
      closeTable()
      i++
      continue
    }

    // 分割线
    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph()
      closeList()
      closeQuote()
      closeTable()
      html.push('<hr>')
      i++
      continue
    }

    // 表格：以 | 开头视为表格行，首行表头 + 分隔行 + 数据行
    if (trimmed.startsWith('|')) {
      const clean = trimmed.replace(/^\|/, '').replace(/\|$/, '').split('|').map((s) => s.trim())
      // 分隔行（|---|）仅用于确认表格，不输出
      if (/^:?-{3,}:?$/.test(clean[0] || '')) {
        i++
        continue
      }
      closeList()
      closeQuote()
      flushParagraph()
      if (!inTable) {
        inTable = true
        tableBuf = []
      }
      tableBuf.push(clean)
      i++
      continue
    }
    closeTable()

    // 标题
    const h = /^(#{1,6})\s+(.*)$/.exec(line)
    if (h) {
      flushParagraph()
      closeList()
      closeQuote()
      const level = h[1].length
      html.push(`<h${level}>${renderInline(h[2])}</h${level}>`)
      i++
      continue
    }

    // 引用
    if (/^>\s?/.test(line)) {
      flushParagraph()
      closeList()
      if (!inQuote) inQuote = true
      quoteBuf.push(renderInline(line.replace(/^>\s?/, '')))
      i++
      continue
    } else {
      closeQuote()
    }

    // 无序列表
    const ul = /^[-*+]\s+(.*)$/.exec(line)
    if (ul) {
      flushParagraph()
      closeQuote()
      if (listType !== 'ul') {
        closeList()
        html.push('<ul>')
        listType = 'ul'
      }
      html.push(`<li>${renderInline(ul[1])}</li>`)
      i++
      continue
    }

    // 有序列表
    const ol = /^\d+[.)]\s+(.*)$/.exec(line)
    if (ol) {
      flushParagraph()
      closeQuote()
      if (listType !== 'ol') {
        closeList()
        html.push('<ol>')
        listType = 'ol'
      }
      html.push(`<li>${renderInline(ol[1])}</li>`)
      i++
      continue
    } else {
      closeList()
    }

    // 普通段落
    paragraph.push(trimmed)
    i++
  }

  flushParagraph()
  closeList()
  closeQuote()
  closeTable()
  if (inCode) html.push(`<pre><code>${escapeHtml(codeBuf.join('\n'))}</code></pre>`)

  return html.join('\n')
}

// 从 Markdown 提取纯文本摘要
export function markdownToText(md, maxLen = 120) {
  if (!md) return ''
  const text = String(md)
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[#>*`~_\-|]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (text.length <= maxLen) return text
  return text.slice(0, maxLen) + '…'
}
