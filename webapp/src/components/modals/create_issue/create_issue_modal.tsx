// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Modal} from 'react-bootstrap';

import CreateIssueForm from './create_issue_form';

type Props = {
    visible: boolean;
    close: () => void;
}

export default function CreateIssueModal(props: Props) {
    if (!props.visible) {
        return null;
    }

    return (
        <Modal
            dialogClassName='modal--scroll'
            show={props.visible}
            onHide={props.close}
            onExited={props.close}
            bsSize='large'
            backdrop='static'
            aria-labelledby='jira-create-issue-modal-title'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title id='jira-create-issue-modal-title'>{'Create Jira Issue'}</Modal.Title>
            </Modal.Header>
            <CreateIssueForm {...props}/>
        </Modal>
    );
}
