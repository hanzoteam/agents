// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const exec = require('child_process').exec;

const path = require('path');

const webpack = require('webpack');

const {transform: formatjsTransform} = require('@formatjs/ts-transformer');

const PLUGIN_ID = require('../plugin.json').id;

// The message id every translated string is filed under. It is a CONTENT hash,
// so it is a function of the string and nothing else — which is what lets the
// extractor (@formatjs/cli, `make i18n-extract`) and the compiler agree without
// sharing state. Both must spell it the SAME WAY, or the bundle asks for ids
// src/i18n/en.json does not hold and every message silently falls back to its
// default, in every locale at once.
//
// `--id-interpolation-pattern` is what the CLI calls it; the TypeScript
// transformer calls the same thing `overrideIdFn` and takes this string form.
const idPattern = '[sha512:contenthash:base64:8]';

// ast precompiles each message, which is a runtime optimization for the bundle
// and is deliberately OFF under test — see jest.config.js, where a precompiled
// message renders as {type, value} objects that React refuses as a child.
const formatjs = (overrideIdFn, ast = true) => formatjsTransform({overrideIdFn, ast});

const NPM_TARGET = process.env.npm_lifecycle_event; //eslint-disable-line no-process-env
const isDev = NPM_TARGET === 'debug' || NPM_TARGET === 'debug:watch';

const plugins = [];
if (NPM_TARGET === 'build:watch' || NPM_TARGET === 'debug:watch') {
    plugins.push({
        apply: (compiler) => {
            compiler.hooks.watchRun.tap('WatchStartPlugin', () => {
                // eslint-disable-next-line no-console
                console.log('Change detected. Rebuilding webapp.');
            });
            compiler.hooks.afterEmit.tap('AfterEmitPlugin', () => {
                exec('cd .. && make deploy-from-watch', (err, stdout, stderr) => {
                    if (stdout) {
                        process.stdout.write(stdout);
                    }
                    if (stderr) {
                        process.stderr.write(stderr);
                    }
                });
            });
        },
    });
}

plugins.push(
    new webpack.ProvidePlugin({
        process: 'process/browser',
    }),
);

const config = {
    entry: [
        './src/index.tsx',
    ],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, 'src'),
        },
        modules: [
            'src',
            'node_modules',
            path.resolve(__dirname),
        ],
        extensions: ['*', '.js', '.jsx', '.ts', '.tsx'],
    },
    module: {
        rules: [
            {
                test: /\.(ts|tsx)$/,
                exclude: /node_modules/,
                use: {
                    loader: 'ts-loader',
                    options: {

                        // TypeScript compiles this project, so the settings that
                        // decide the output are tsconfig.json's and there is no
                        // second place they can disagree from. `jsx: react` there
                        // IS the classic runtime: it emits React.createElement,
                        // which is what webpack's `react` external supplies. The
                        // automatic runtime would emit imports from
                        // react/jsx-runtime, a module the host does not hand over
                        // — that shipped once and every agent post died in the
                        // error boundary on `jsxDEV is not a function`.
                        //
                        // Types are checked by `npm run check-types` over the whole
                        // project, not per file inside the bundler — which is what
                        // the pipeline this replaced did, since Babel never checked
                        // types at all. Checking here instead would make a type
                        // error somewhere else a BUILD failure for whoever touched
                        // an unrelated file, reported as a webpack stack trace.
                        transpileOnly: true,

                        // tsconfig keeps `noEmit` for that check-types run; a loader
                        // exists to emit, so it says so here rather than the project
                        // dropping the flag and `tsc` scattering .js beside every
                        // source file.
                        compilerOptions: {noEmit: false},
                        getCustomTransformers: () => ({before: [formatjs(idPattern)]}),
                    },
                },
            },
            {
                test: /\.json$/,
                type: 'json',
            },
            {
                test: /\.(png|eot|tiff|svg|woff2|woff|ttf|gif|mp3|jpg|jpeg)$/,
                type: 'asset/inline',
            },
        ],
    },
    externals: {
        react: 'React',
        'react-dom': 'ReactDOM',
        redux: 'Redux',
        'react-redux': 'ReactRedux',
        'prop-types': 'PropTypes',
        'react-intl': 'ReactIntl',
        'react-bootstrap': 'ReactBootstrap',
        'react-router-dom': 'ReactRouterDom',
    },
    output: {
        devtoolNamespace: PLUGIN_ID,
        path: path.join(__dirname, '/dist'),
        publicPath: '/',
        filename: 'main.js',
    },
    mode: isDev ? 'development' : 'production',
    devtool: isDev ? 'source-map' : false,
    plugins,
};

module.exports = config;
