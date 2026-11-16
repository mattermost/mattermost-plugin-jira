// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef, useState} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Checkbox} from '@mattermost/compass-ui/components/checkbox';
import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import Modal from 'components/modal';

import {Z_INDEX_CONFIRM_MODAL} from 'utils/styles';

export type Props = {
    show: boolean;
    title: React.ReactNode;
    message: React.ReactNode;
    confirmButtonText: React.ReactNode;
    cancelButtonText?: React.ReactNode;
    destructive?: boolean;
    showCheckbox?: boolean;
    checkboxText?: string;

    /** Receives whether the checkbox is checked. */
    onConfirm: (checked: boolean) => void;
    onCancel: (checked: boolean) => void;
    onExited?: () => void;
    hideCancel?: boolean;
};

export default function ConfirmModal(props: Props) {
    const [checked, setChecked] = useState(false);
    const confirmRef = useRef<HTMLButtonElement>(null);

    const handleCancel = () => props.onCancel(checked);
    const handleConfirm = () => props.onConfirm(checked);

    const footer = (
        <>
            {!props.hideCancel && (
                <Button
                    id='cancelModalButton'
                    emphasis='tertiary'
                    onClick={handleCancel}
                >
                    {props.cancelButtonText || 'Cancel'}
                </Button>
            )}
            <Button
                id='confirmModalButton'
                ref={confirmRef}
                emphasis='primary'
                destructive={props.destructive}
                onClick={handleConfirm}
            >
                {props.confirmButtonText}
            </Button>
        </>
    );

    return (
        <ModalOverlay
            open={props.show}
            onClose={handleCancel}
            onExited={props.onExited}
            zIndex={Z_INDEX_CONFIRM_MODAL}
        >
            <Modal
                id='confirmModal'
                open={props.show}
                size='small'
                title={props.title}
                headerDivider={false}
                footerDivider={false}
                onClose={handleCancel}
                initialFocusRef={confirmRef}
                footer={footer}
            >
                {props.message}
                {props.showCheckbox && (
                    <Checkbox
                        checked={checked}
                        onChange={(e) => setChecked(e.target.checked)}
                    >
                        {props.checkboxText}
                    </Checkbox>
                )}
            </Modal>
        </ModalOverlay>
    );
}
