// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Modal} from 'react-bootstrap';

import {Theme} from 'mattermost-redux/types/preferences';

import AttachCommentToIssueForm from './attach_comment_form';

type Props = {
    visible: boolean;
    theme: Theme;
    close: () => void;
}

export default function AttachCommentToIssueModal(props: Props) {
    const {visible} = props;
    if (!visible) {
        return null;
    }

    return (
        <Modal
            dialogClassName='modal--scroll'
            show={visible}
            onHide={props.close}
            onExited={props.close}
            bsSize='large'
            backdrop='static'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>
                    {'Attach Message to Jira Issue'}
                </Modal.Title>
            </Modal.Header>
            <AttachCommentToIssueForm {...props}/>
        </Modal>
    );
}
