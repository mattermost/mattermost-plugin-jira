// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Instance} from 'types/model';

export type Props = {
    theme: {};
    visible: boolean;
    connectedInstances: Instance[];
    closeModal: () => void;
    disconnectUser: (instanceID: string) => Promise<{data?: {}; error?: Error}>;
    sendEphemeralPost: (message: string) => void;
};
