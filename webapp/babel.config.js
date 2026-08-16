// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const config = {
    presets: [
        ['@babel/preset-env', {
            targets: {
                chrome: 110,
                firefox: 102,
                edge: 110,
                safari: 16.2,
            },
            modules: false,
            debug: false,
            shippedProposals: true,
        }],
        ['@babel/preset-react'],

        // Babel 8 removed .allExtensions/.isTSX: JSX is detected from the file
        // extension, and every file here is already .ts or .tsx.
        ['@babel/typescript'],
    ],
    plugins: [
        ['polyfill-corejs3', {method: 'usage-global'}],
        [
            'formatjs',
            {
                idInterpolationPattern: '[sha512:contenthash:base64:8]',
                ast: true,
            },
        ],
    ],
};

// Jest runs this same pipeline (jest.config.js sets no `transform`, so babel-jest
// reads this file). Two things differ under test, and both are deep-copied rather
// than shared: the arrays above are the objects webpack builds with, so mutating
// them in place would change the shipped bundle too.
//
//   modules 'auto'  — jest needs CommonJS; the browser build wants ESM left alone.
//   formatjs ast    — precompiling messages to an AST is a runtime optimization
//                     for the bundle. Under test it makes a message render as
//                     `{type, value}` objects, which React refuses as a child
//                     ("Objects are not valid as a React child"), so tests assert
//                     against structure instead of the string a user reads.
const testConfig = JSON.parse(JSON.stringify({presets: config.presets, plugins: config.plugins}));
testConfig.presets[0][1].modules = 'auto';
for (const plugin of testConfig.plugins) {
    if (Array.isArray(plugin) && plugin[0] === 'formatjs') {
        plugin[1].ast = false;
    }
}
config.env = {test: testConfig};

module.exports = config;
