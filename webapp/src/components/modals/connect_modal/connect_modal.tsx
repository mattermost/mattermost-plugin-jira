// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';

import ConnectModalForm from './connect_modal_form';
import {Props} from './props';

export default function ConnectModal(props: Props) {
    return (
        <ModalOverlay
            open={props.visible}
            zIndex={Z_INDEX_MODAL}
        >
            <ConnectModalForm {...props}/>
        </ModalOverlay>
    );
}
