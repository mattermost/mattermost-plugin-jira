// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

let counter = 0;

export default function uniqueId(prefix: string): string {
    counter += 1;
    return `${prefix}-${counter}`;
}
