// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {isMinimumServerVersion} from 'mattermost-redux/utils/helpers';

const userAgent = window.navigator.userAgent;

export function isDesktopApp() {
    return userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
}

export function getDesktopAppVersion() {
    if (window.desktop && window.desktop.version) {
        return window.desktop.version;
    }
    return null;
}

export function isMinimumDesktopAppVersion(minMajorVersion, minMinorVersion, minDotVersion) {
    const currentVersion = getDesktopAppVersion();
    if (!currentVersion) {
        return false;
    }

    return isMinimumServerVersion(currentVersion, minMajorVersion, minMinorVersion, minDotVersion);
}
