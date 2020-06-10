// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Instance} from 'types/model';

export type Props = {
    theme: {};
    visible: boolean;
    connectedInstances: Instance[];
    closeModal: () => void;
    redirectConnect: (instanceID: string) => void;
};
