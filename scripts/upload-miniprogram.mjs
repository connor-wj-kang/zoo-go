#!/usr/bin/env node
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import {fileURLToPath} from 'node:url'
import ci from 'miniprogram-ci'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const projectPath = path.join(root, 'miniprogram')
const projectConfig = JSON.parse(fs.readFileSync(path.join(projectPath, 'project.config.json'), 'utf8'))

const appid = process.env.WECHAT_APPID || projectConfig.appid
const privateKey = process.env.WECHAT_MINIPROGRAM_PRIVATE_KEY
const version = process.env.MINIPROGRAM_VERSION || new Date().toISOString().replace(/[-:TZ.]/g, '').slice(0, 14)
const desc = process.env.MINIPROGRAM_DESC || 'ci upload'
const robot = Number(process.env.MINIPROGRAM_ROBOT || '1')

if (!appid) {
    throw new Error('missing WECHAT_APPID / project.config.json appid')
}
if (!privateKey) {
    throw new Error('missing WECHAT_MINIPROGRAM_PRIVATE_KEY')
}

const keyPath = path.join(os.tmpdir(), `private.${appid}.key`)
fs.writeFileSync(keyPath, privateKey.replace(/\\n/g, '\n'), {mode: 0o600})

const setting = projectConfig.setting || {}
const project = new ci.Project({
    appid,
    type: 'miniProgram',
    projectPath,
    privateKeyPath: keyPath,
    ignores: ['node_modules/**/*', '**/.*'],
})

try {
    const result = await ci.upload({
        project,
        version,
        desc,
        robot,
        setting: {
            es6: setting.es6 !== false,
            es7: true,
            minify: setting.minified !== false,
            codeProtect: false,
            minifyJS: setting.minified !== false,
            minifyWXML: setting.minifyWXML !== false,
            minifyWXSS: setting.minifyWXSS !== false,
            autoPrefixWXSS: setting.postcss !== false,
        },
        onProgressUpdate: console.log,
    })
    console.log(JSON.stringify({appid, version, desc, robot, result}, null, 2))
} finally {
    fs.rmSync(keyPath, {force: true})
}
