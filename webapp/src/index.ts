// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from './manifest';
import Plugin from './plugin';

window.registerPlugin(manifest.id, new Plugin());
