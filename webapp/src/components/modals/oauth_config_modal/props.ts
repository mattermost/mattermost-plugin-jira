// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {APIResponse, OAuthConfig, Instance} from 'types/model';

export type Props = {
    theme: {};
    visible: boolean;
    installedInstances: Instance[];
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    closeModal: () => void;
    configure: (config: OAuthConfig) => Promise<APIResponse<{}>>;
};
