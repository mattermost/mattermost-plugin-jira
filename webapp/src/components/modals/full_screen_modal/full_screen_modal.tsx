// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {CSSTransition} from 'react-transition-group';

import CloseIcon from './close_icon';

// This must be on sync with the animation time in ./full_screen_modal.scss
const ANIMATION_DURATION = 100;

type Props = {
    show: boolean;
    children: React.ReactNode;
    onClose: () => void;
};

export default class FullScreenModal extends React.Component<Props> {
    componentDidMount() {
        document.addEventListener('keydown', this.handleKeypress);
    }

    componentWillUnmount() {
        document.removeEventListener('keydown', this.handleKeypress);
    }

    handleKeypress = (e: KeyboardEvent) => {
        if (e.key === 'Escape' && this.props.show) {
            this.close();
        }
    };

    close = () => {
        this.props.onClose();
    };

    render() {
        return (
            <CSSTransition
                in={this.props.show}
                classNames='FullScreenModal'
                mountOnEnter={true}
                unmountOnExit={true}
                timeout={ANIMATION_DURATION}
                appear={true}
            >
                <div className='FullScreenModal FullScreenModal--compact'>
                    <CloseIcon
                        className='close-x'
                        onClick={this.close}
                    />
                    {this.props.children}
                </div>
            </CSSTransition>
        );
    }
}
