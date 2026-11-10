// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';

import AttachCommentToIssueForm from './attach_comment_form';

type Props = {
    visible: boolean;
    close: () => void;
}

export default function AttachCommentToIssueModal(props: Props) {
    return (
        <ModalOverlay
            open={props.visible}
            zIndex={Z_INDEX_MODAL}
        >
            <AttachCommentToIssueForm {...props}/>
        </ModalOverlay>
    );
}
