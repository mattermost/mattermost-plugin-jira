// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Trims Atlassian's published Jira Cloud OpenAPI specs down to the operations fake-jira serves.
//   node update_specs.mjs          rewrite specs/*.json from upstream
//   node update_specs.mjs --check  exit 1 if upstream drifted from the committed copies

/* eslint-disable no-console */

import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const dir = path.dirname(fileURLToPath(import.meta.url));
const specsDir = path.join(dir, 'specs');
const check = process.argv.includes('--check');

const manifest = JSON.parse(await fs.readFile(path.join(dir, 'operations.json'), 'utf8'));
let drifted = false;

for (const [name, {url, operations}] of Object.entries(manifest)) {
    const res = await fetch(url);
    if (!res.ok) {
        throw new Error(`Failed to download ${url}: ${res.status}`);
    }
    const trimmed = trim(await res.json(), operations, name);
    const outPath = path.join(specsDir, `${name}.json`);
    const next = `${JSON.stringify(trimmed, null, 2)}\n`;

    if (check) {
        const current = await fs.readFile(outPath, 'utf8').catch(() => '');
        if (current !== next) {
            drifted = true;
            console.error(`${name}: upstream spec changed for the operations fake-jira serves (${url})`);
        }
        continue;
    }

    await fs.mkdir(specsDir, {recursive: true});
    await fs.writeFile(outPath, next);
    console.log(`${name}: ${operations.length} operations, ${Object.keys(trimmed.components.schemas).length} schemas`);
}

if (drifted) {
    console.error('Run `npm run contract:update`, review the diff, and update fake-jira to match.');
    process.exit(1);
}

function trim(spec, operations, name) {
    const paths = {};
    for (const operation of operations) {
        const [method, opPath] = operation.split(' ');
        const op = spec.paths[opPath]?.[method.toLowerCase()];
        if (!op) {
            throw new Error(`${name}: ${operation} is not in the upstream spec`);
        }
        paths[opPath] = {...paths[opPath], [method.toLowerCase()]: {requestBody: op.requestBody, responses: op.responses}};
    }

    const schemas = {};
    const queue = collectRefs(paths);
    while (queue.length) {
        const schemaName = queue.pop();
        if (schemas[schemaName]) {
            continue;
        }
        schemas[schemaName] = spec.components.schemas[schemaName];
        queue.push(...collectRefs(schemas[schemaName]));
    }

    return {
        openapi: spec.openapi,
        // info.version embeds an upstream build hash that changes on every deploy, so it's omitted to keep --check meaningful.
        info: {title: spec.info.title},
        paths,
        components: {schemas: Object.fromEntries(Object.entries(schemas).sort(([a], [b]) => a.localeCompare(b)))},
    };
}

function collectRefs(node, found = []) {
    if (Array.isArray(node)) {
        node.forEach((item) => collectRefs(item, found));
    } else if (node && typeof node === 'object') {
        for (const [key, value] of Object.entries(node)) {
            if (key === '$ref' && typeof value === 'string') {
                found.push(value.replace('#/components/schemas/', ''));
            } else if (key !== 'example' && key !== 'examples') {
                collectRefs(value, found);
            }
        }
    }
    return found;
}
