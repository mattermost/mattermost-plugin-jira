// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useRef} from 'react';

// ModalOverlay keeps its children mounted through the exit transition, so a form would
// otherwise be reused on the next open and see cleared props while closing.
export default function useModalSession<P extends {visible: boolean}>(props: P): {key: number; props: P} {
    const session = useRef(0);
    const wasVisible = useRef(false);
    const lastProps = useRef(props);

    if (props.visible) {
        if (!wasVisible.current) {
            session.current += 1;
        }
        lastProps.current = props;
    }
    wasVisible.current = props.visible;

    return {key: session.current, props: {...lastProps.current, visible: props.visible}};
}
