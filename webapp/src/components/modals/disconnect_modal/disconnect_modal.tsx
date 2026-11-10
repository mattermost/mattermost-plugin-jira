// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';

import DisconnectModalForm from './disconnect_modal_form';
import {Props} from './props';

export default function DisconnectModal(props: Props) {
    return (
        <ModalOverlay
            open={props.visible}
            zIndex={Z_INDEX_MODAL}
        >
            <DisconnectModalForm {...props}/>
        </ModalOverlay>
    );
}
