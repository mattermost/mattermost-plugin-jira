// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useRef} from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {Z_INDEX_MODAL} from 'utils/styles';
import useModalSession from 'utils/use_modal_session';

import CreateIssueForm from './create_issue_form';

type Props = {
    visible: boolean;
    close: () => void;
}

export default function CreateIssueModal(props: Props) {
    const session = useModalSession(props);
    const form = useRef<CreateIssueForm>(null);

    return (
        <ModalOverlay
            open={props.visible}
            onClose={() => (form.current ? form.current.requestClose() : props.close())}
            zIndex={Z_INDEX_MODAL}
        >
            <CreateIssueForm
                ref={form}
                key={session.key}
                {...session.props}
            />
        </ModalOverlay>
    );
}
