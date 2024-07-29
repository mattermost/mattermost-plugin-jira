// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Modal} from 'react-bootstrap';

import ConnectModalForm from './connect_modal_form';
import {Props} from './props';

export default function ConnectModal(props: Props) {
    const {visible} = props;
    let content;
    if (visible) {
        content = (
            <ConnectModalForm {...props}/>
        );
    }

    return (
        <Modal
            dialogClassName='modal--scroll'
            show={visible}
            onHide={props.closeModal}
            onExited={props.closeModal}
            bsSize='large'
            backdrop='static'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>
                    {'Connect to Jira'}
                </Modal.Title>
            </Modal.Header>
            {content}
        </Modal>
    );
}
