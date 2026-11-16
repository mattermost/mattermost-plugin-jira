// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import '@mattermost/compass-ui/styles';

import manifest from './manifest';
import Plugin from './plugin';

window.registerPlugin(manifest.id, new Plugin());
