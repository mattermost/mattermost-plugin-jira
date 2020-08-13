// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
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
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>{'Create Jira Issue'}</Modal.Title>
            </Modal.Header>
            <CreateIssueForm {...props}/>
        </Modal>
    );
}
