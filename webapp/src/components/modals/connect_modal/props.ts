// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Theme} from 'mattermost-redux/types/preferences';

import {Instance} from 'types/model';

export type Props = {
    theme: Theme;
    visible: boolean;
    installedInstances: Instance[];
    connectedInstances: Instance[];
    closeModal: () => void;
    redirectConnect: (instanceID: string) => void;
};
