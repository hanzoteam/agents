// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// The same compiler and the same message-id function webpack builds with, so a
// test exercises the code that ships rather than a second translation of it. It
// used to be babel-jest reading a babel.config.js that carried an `env.test`
// section, which is the shape that let the project compile its own source two
// ways; there is one way now, and it is TypeScript.
//
// Types are checked by `npm run check-types` over the whole project, not per file
// here — the same split the bundler makes, and the reason both use transpile-only.
module.exports = {
    testEnvironment: 'jsdom',
    transform: {
        '^.+\\.tsx?$': ['ts-jest', {
            astTransformers: {
                before: [{

                    // The extension is part of the subpath: the package's exports
                    // map keys this entry as "./ts-jest-integration.js", and a
                    // bare specifier is ERR_PACKAGE_PATH_NOT_EXPORTED.
                    path: require.resolve('@formatjs/ts-transformer/ts-jest-integration.js'),
                    options: {

                        // Identical to webpack.config.js. A test that compiled ids
                        // differently would pass against messages the bundle never
                        // asks for.
                        overrideIdFn: '[sha512:contenthash:base64:8]',

                        // ast is the one deliberate difference from the shipped
                        // build. Precompiling a message yields {type, value}
                        // objects, which React refuses as a child ("Objects are not
                        // valid as a React child"), so a test asserting on rendered
                        // text would have to assert on structure instead.
                        ast: false,
                    },
                }],
            },
        }],
    },
    moduleNameMapper: {

        // Asset mappings must precede the path aliases below: moduleNameMapper
        // is evaluated in order and '^@/(.*)$' would otherwise match asset
        // imports first.
        '\\.(svg|png|jpg|jpeg|gif|webp)$': '<rootDir>/tests/svg_mock.js',
        '^@/(.*)$': '<rootDir>/src/$1',
    },
    setupFilesAfterEnv: ['<rootDir>/tests/setup.tsx'],
};
