// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useRef} from 'react';

import {Modal as CompassModal, ModalProps} from '@mattermost/compass-ui/components/modal';

import {useTooltip} from 'components/with_tooltip';

type StackEntry = {close: () => void};

const openModals: StackEntry[] = [];

// The webapp's search bar calls preventDefault on every Escape at the document
// level before Compass's own listener runs (mattermost/compass-design#247), so
// Escape is handled on the dialog root, after inner widgets have had their turn.
// Keys from portaled popups (date picker calendar) are left to their own handlers.
// When focus has fallen back to the body, the topmost modal closes instead.
export default function Modal({onKeyDown, backLabel = 'Back', backButtonProps, ...props}: ModalProps) {
    const backTooltip = useTooltip(backLabel);
    const latest = useRef(props);
    latest.current = props;

    useEffect(() => {
        const entry: StackEntry = {
            close: () => {
                if (latest.current.open !== false) {
                    latest.current.onClose?.();
                }
            },
        };
        openModals.push(entry);

        const handleBodyEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && e.target === document.body && openModals[openModals.length - 1] === entry) {
                e.preventDefault();
                entry.close();
            }
        };
        document.addEventListener('keydown', handleBodyEscape, true);

        return () => {
            openModals.splice(openModals.indexOf(entry), 1);
            document.removeEventListener('keydown', handleBodyEscape, true);
        };
    }, []);

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
        onKeyDown?.(e);
        const fromDialog = e.currentTarget.contains(e.target as Node);
        if (e.key === 'Escape' && fromDialog && !e.defaultPrevented && props.onClose) {
            e.preventDefault();
            props.onClose();
        }
    };

    return (
        <>
            <CompassModal
                {...props}
                backLabel={backLabel}
                backButtonProps={{...backButtonProps, ...backTooltip.triggerProps}}
                onKeyDown={handleKeyDown}
            />
            {props.showBackButton && backTooltip.tooltip}
        </>
    );
}
