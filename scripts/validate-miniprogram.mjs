import fs from 'node:fs'
import path from 'node:path'
import {fileURLToPath} from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'miniprogram')
const app = JSON.parse(fs.readFileSync(path.join(root, 'app.json'), 'utf8'))
const missing = []

for (const page of app.pages) {
    for (const extension of ['.js', '.json', '.wxml', '.wxss']) {
        if (!fs.existsSync(path.join(root, page + extension))) {
            missing.push(page + extension)
        }
    }
}
if (missing.length) {
    throw new Error(`missing page files: ${missing.join(', ')}`)
}

const wxmlFiles = []
const walk = (directory) => {
    for (const entry of fs.readdirSync(directory, {withFileTypes: true})) {
        const fullPath = path.join(directory, entry.name)
        if (entry.isDirectory()) {
            walk(fullPath)
        } else if (entry.name.endsWith('.wxml')) {
            wxmlFiles.push(fullPath)
        }
    }
}
walk(root)

const voidTags = new Set(['image', 'input', 'textarea', 'switch'])
for (const file of wxmlFiles) {
    const source = fs.readFileSync(file, 'utf8').replace(/{{[\s\S]*?}}/g, 'EXPR')
    const stack = []
    const tagPattern = /<\/?([a-zA-Z][\w-]*)(?:\s[^<>]*?)?(\/?)>/g
    let match
    while ((match = tagPattern.exec(source))) {
        const closing = match[0].startsWith('</')
        const name = match[1]
        const selfClosing = match[2] === '/' || voidTags.has(name)
        if (closing) {
            const top = stack.pop()
            if (top !== name) {
                throw new Error(`${file}: expected </${top}> but found </${name}>`)
            }
        } else if (!selfClosing) {
            stack.push(name)
        }
    }
    if (stack.length) {
        throw new Error(`${file}: unclosed tags ${stack.join(', ')}`)
    }
}

for (const file of fs.readdirSync(path.join(root, 'components'), {withFileTypes: true})) {
    if (!file.isDirectory()) {
        continue
    }
    for (const extension of ['.js', '.json', '.wxml', '.wxss']) {
        const fullPath = path.join(root, 'components', file.name, 'index' + extension)
        if (!fs.existsSync(fullPath)) {
            throw new Error(`missing component file: ${fullPath}`)
        }
    }
}

console.log(`Mini program checks OK: ${app.pages.length} pages, ${wxmlFiles.length} WXML files`)
