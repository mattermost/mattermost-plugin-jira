// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

import {Instance, SelectOption} from 'types/model';

import SelectSetting from 'components/select_setting';

import {Props} from './props';
const FORM_ID = 'jira-connect-modal-form';

export type State = {
    submitting: boolean;
    error: string;
    selectedInstance: string;
};

export default class ConnectModalForm extends PureComponent<Props, State> {
    state = {
        submitting: false,
        error: '',
        selectedInstance: '',
    };

    submit = async (e: React.FormEvent<HTMLFormElement>) => {
        if (e.preventDefault) {
            e.preventDefault();
        }

        const selectedInstance = this.state.selectedInstance;
        if (!selectedInstance) {
            this.setState({error: 'Please select a Jira instance'});
            return;
        }

        if (this.isAlreadyConnectedToInstance(this.state.selectedInstance)) {
            this.setState({error: 'You are already connected to this Jira instance.'});
            return;
        }

        this.props.closeModal();
        this.props.redirectConnect(this.state.selectedInstance);
    };

    isAlreadyConnectedToInstance = (instanceID: string): boolean => {
        return Boolean(this.props.connectedInstances.find((instance) => instance.instance_id === instanceID));
    };

    closeModal = () => {
        this.props.closeModal();
    };

    handleInstanceChoice = (_: string, instanceID: string) => {
        if (instanceID === this.state.selectedInstance) {
            return;
        }

        let error = '';
        if (instanceID && this.isAlreadyConnectedToInstance(instanceID)) {
            error = 'You are already connected to this Jira instance.';
        }

        this.setState({selectedInstance: instanceID, error});
    };

    render(): JSX.Element {
        const {selectedInstance} = this.state;

        const options: SelectOption[] = this.props.installedInstances.map((instance: Instance) => (
            {label: instance.alias || instance.instance_id, value: instance.instance_id}
        ));
        const instanceSelector = (
            <SelectSetting
                name='instance'
                label='Jira Instance'
                options={options}
                onChange={this.handleInstanceChoice}
                value={options.find((opt) => opt.value === this.state.selectedInstance)}
            />
        );

        const disableSubmit = !selectedInstance || this.isAlreadyConnectedToInstance(selectedInstance);
        const footer = (
            <>
                <Button
                    emphasis='tertiary'
                    onClick={this.closeModal}
                >
                    {'Cancel'}
                </Button>
                <Button
                    type='submit'
                    form={FORM_ID}
                    emphasis='primary'
                    disabled={disableSubmit}
                    loading={this.state.submitting}
                >
                    {'Connect'}
                </Button>
            </>
        );

        return (
            <Modal
                open={this.props.visible}
                size='small'
                title='Connect to Jira'
                onClose={this.closeModal}
                footer={footer}
            >
                <form
                    id={FORM_ID}
                    role='form'
                    onSubmit={this.submit}
                >
                    {this.state.error && (
                        <SectionNotice
                            className='jira-form-notice'
                            role='alert'
                            type='danger'
                            title={this.state.error}
                        />
                    )}
                    {instanceSelector}
                </form>
            </Modal>
        );
    }
}
