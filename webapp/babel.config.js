// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const config = {
    presets: [
        ['@babel/preset-env', {
            targets: {
                chrome: 66,
                firefox: 60,
                edge: 42,
                safari: 12,
            },
            modules: false,
            corejs: 3,
            debug: false,
            useBuiltIns: 'usage',
            shippedProposals: true,
        }],
        ['@babel/preset-react', {
            runtime: 'automatic',
        }],
        ['@babel/typescript', {
            allExtensions: true,
            isTSX: true,
        }],
    ],
    plugins: [
        '@babel/plugin-transform-class-properties',
        '@babel/plugin-syntax-dynamic-import',
        '@babel/plugin-transform-object-rest-spread',
        'babel-plugin-typescript-to-proptypes',
    ],
};

config.env = {
    test: {
        presets: [
            ['@babel/preset-env', {
                targets: {
                    chrome: 66,
                    firefox: 60,
                    edge: 42,
                    safari: 12,
                },
                modules: 'auto',
                corejs: 3,
                debug: false,
                useBuiltIns: 'usage',
                shippedProposals: true,
            }],
            ['@babel/preset-react', {
                runtime: 'automatic',
            }],
            ['@babel/typescript', {
                allExtensions: true,
                isTSX: true,
            }],
        ],
        plugins: [
            '@babel/plugin-transform-class-properties',
            '@babel/plugin-syntax-dynamic-import',
            '@babel/plugin-transform-object-rest-spread',
            'babel-plugin-typescript-to-proptypes',
        ],
    },
};

module.exports = config;
