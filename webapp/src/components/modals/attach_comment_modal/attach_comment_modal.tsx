// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';
import useModalSession from 'utils/use_modal_session';

import AttachCommentToIssueForm from './attach_comment_form';

type Props = {
    visible: boolean;
    close: () => void;
}

export default function AttachCommentToIssueModal(props: Props) {
    const session = useModalSession(props);

    return (
        <ModalOverlay
            open={props.visible}
            onClose={props.close}
            zIndex={Z_INDEX_MODAL}
        >
            <AttachCommentToIssueForm
                key={session.key}
                {...session.props}
            />
        </ModalOverlay>
    );
}
