import {Instance} from 'types/model';

export type Props = {
    theme: {};
    visible: boolean;
    closeModal: () => void;
    disconnectUser: (instanceID: string) => Promise<{data?: {}; error?: Error}>;
    sendEphemeralPost: (message: string) => void;
};
