// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

module.exports = {
    testEnvironment: 'jsdom',

    // No `transform` override: jest's default is babel-jest, which reads
    // babel.config.js — the same pipeline webpack builds with. babel.config.js
    // has always carried an `env.test` ("Jest needs module transformation") that
    // an explicit ts-jest transform made dead code, so the project was compiling
    // its own source two different ways and only one of them shipped.
    //
    // Types are still checked, by `npm run check-types` (tsc --noEmit) over the
    // whole project, rather than per-file inside the test runner.
    moduleNameMapper: {

        // Asset mappings must precede the path aliases below: moduleNameMapper
        // is evaluated in order and '^src/(.*)$' would otherwise match asset
        // imports like 'src/../../assets/bot_icon.png' first.
        '\\.(svg|png|jpg|jpeg|gif|webp)$': '<rootDir>/tests/svg_mock.js',
        '^@/(.*)$': '<rootDir>/src/$1',
    },
    setupFilesAfterEnv: ['<rootDir>/tests/setup.tsx'],
};
