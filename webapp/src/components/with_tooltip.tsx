// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import ReactDOM from 'react-dom';

import {Tooltip} from '@mattermost/compass-ui/components/tooltip';

import {Z_INDEX_MENU_PORTAL} from 'utils/styles';

import './with_tooltip.scss';

const TOOLTIP_GAP = 4;

export function useTooltip(label: string) {
    const [anchor, setAnchor] = useState<DOMRect | null>(null);

    const show = (e: React.SyntheticEvent<HTMLElement>) => setAnchor(e.currentTarget.getBoundingClientRect());
    const hide = () => setAnchor(null);

    const tooltip = anchor && ReactDOM.createPortal(
        <div
            className='jira-tooltip'
            role='presentation'
            data-testid='jira-tooltip'
            style={{
                top: anchor.top - TOOLTIP_GAP,
                left: anchor.left + (anchor.width / 2),
                zIndex: Z_INDEX_MENU_PORTAL,
            }}
        >
            <Tooltip
                label={label}
                arrow='bottom'
            />
        </div>,
        document.body,
    );

    return {
        triggerProps: {onMouseEnter: show, onMouseLeave: hide, onFocus: show, onBlur: hide},
        tooltip,
    };
}

type Props = {
    label: string;
    children: React.ReactNode;
};

export default function WithTooltip({label, children}: Props) {
    const {triggerProps, tooltip} = useTooltip(label);

    return (
        <span
            className='jira-tooltip-anchor'
            {...triggerProps}
        >
            {children}
            {tooltip}
        </span>
    );
}
