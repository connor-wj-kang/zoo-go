// ESLint 开关: 设置为 false 禁用所有检查，设置为 true 启用检查，默认关闭即可
const ESLINT_ENABLED = false;

// 如果禁用，返回最小化配置
if (!ESLINT_ENABLED) {
    module.exports = {
        ignorePatterns: ['**/*'],
    };
} else {
    // 启用时的完整配置
    module.exports = {
        parser: '@babel/eslint-parser',
        parserOptions: {
            requireConfigFile: false,
            babelOptions: {
                presets: [
                    [require.resolve('@babel/preset-react'), {runtime: 'automatic'}],
                ],
            },
        },
        extends: [
            '@ecomfe/eslint-config',
            '@ecomfe/eslint-config/baidu/default',
            '@ecomfe/eslint-config/baidu/defect',
            '@ecomfe/eslint-config/react',
        ],
    };
};
