// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import {injectIntl} from 'react-intl';

import {ReactSelectOption, Instance} from 'types/model';

import {getModalStyles} from 'utils/styles';

import FormButton from 'components/form_button';
import ReactSelectSetting from 'components/react_select_setting';

import {Props} from './props';
export type State = {
    submitting: boolean;
    error: string;
    selectedInstance: string;
};

export class DisconnectModalForm extends PureComponent<Props, State> {
    state = {
        submitting: false,
        error: '',
        selectedInstance: '',
    };

    submit = async (e) => {
        const {formatMessage} = this.props.intl;
        if (e.preventDefault) {
            e.preventDefault();
        }

        const selectedInstance = this.state.selectedInstance;
        if (!selectedInstance) {
            this.setState({error: formatMessage({defaultMessage: 'Please select a Jira instance'})});
            return;
        }

        this.props.disconnectUser(selectedInstance).then(({error}) => {
            if (error) {
                this.setState({error: error.toString()});
            } else {
                this.props.sendEphemeralPost(formatMessage({defaultMessage: 'Successfully disconnected from Jira instance {selectedInstance}'}, {selectedInstance}));
                this.props.closeModal();
            }
        });
    }

    closeModal = (e) => {
        this.props.closeModal();
    }

    handleInstanceChoice = (_: string, instanceID: string) => {
        if (instanceID === this.state.selectedInstance) {
            return;
        }

        this.setState({selectedInstance: instanceID, error: ''});
    }

    render(): JSX.Element {
        const {formatMessage} = this.props.intl;
        const style = getModalStyles(this.props.theme);

        const options: ReactSelectOption[] = this.props.connectedInstances.map((instance: Instance) => (
            {label: instance.instance_id, value: instance.instance_id}
        ));
        const instanceSelector = (
            <ReactSelectSetting
                options={options}
                theme={this.props.theme}
                onChange={this.handleInstanceChoice}
                value={options.find((opt) => opt.value === this.state.selectedInstance)}
            />
        );

        const footer = (
            <React.Fragment>
                <FormButton
                    type='button'
                    btnClass='btn-link'
                    defaultMessage={formatMessage({defaultMessage: 'Cancel'})}
                    onClick={this.closeModal}
                />
                <FormButton
                    type='submit'
                    btnClass='btn btn-primary'
                    defaultMessage={formatMessage({defaultMessage: 'Disconnect'})}
                    disabled={!this.state.selectedInstance}
                    saving={this.state.submitting}
                />
                <p className={'error-text'}>
                    {this.state.error}
                </p>
            </React.Fragment>
        );

        return (
            <form
                role='form'
                onSubmit={this.submit}
            >
                <Modal.Body
                    style={style.modalBody}
                >
                    {instanceSelector}
                </Modal.Body>
                <Modal.Footer style={style.modalFooter}>
                    {footer}
                </Modal.Footer>
            </form>
        );
    }
}

export default injectIntl(DisconnectModalForm);
