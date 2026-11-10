// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';

import CreateIssueForm from './create_issue_form';

type Props = {
    visible: boolean;
    close: () => void;
}

export default function CreateIssueModal(props: Props) {
    return (
        <ModalOverlay
            open={props.visible}
            zIndex={Z_INDEX_MODAL}
        >
            <CreateIssueForm {...props}/>
        </ModalOverlay>
    );
}
