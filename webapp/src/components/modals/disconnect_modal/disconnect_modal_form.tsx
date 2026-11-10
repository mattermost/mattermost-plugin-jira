// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

import {Instance, SelectOption} from 'types/model';

import SelectSetting from 'components/select_setting';

import {Props} from './props';
const FORM_ID = 'jira-disconnect-modal-form';

export type State = {
    submitting: boolean;
    error: string;
    selectedInstance: string;
};

export default class DisconnectModalForm extends PureComponent<Props, State> {
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

        this.props.disconnectUser(selectedInstance).then(({error}) => {
            if (error) {
                this.setState({error: error.toString()});
            } else {
                this.props.sendEphemeralPost(`Successfully disconnected from Jira instance ${selectedInstance}`);
                this.props.closeModal();
            }
        });
    };

    closeModal = () => {
        this.props.closeModal();
    };

    handleInstanceChoice = (_: string, instanceID: string) => {
        if (instanceID === this.state.selectedInstance) {
            return;
        }

        this.setState({selectedInstance: instanceID, error: ''});
    };

    render(): JSX.Element {
        const options: SelectOption[] = this.props.connectedInstances.map((instance: Instance) => (
            {label: instance.instance_id, value: instance.instance_id}
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
                    destructive={true}
                    disabled={!this.state.selectedInstance}
                    loading={this.state.submitting}
                >
                    {'Disconnect'}
                </Button>
            </>
        );

        return (
            <Modal
                open={this.props.visible}
                size='small'
                title='Disconnect from Jira'
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
