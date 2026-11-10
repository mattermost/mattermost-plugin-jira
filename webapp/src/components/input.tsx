// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {TextArea} from '@mattermost/compass-ui/components/text-area';
import {TextInput} from '@mattermost/compass-ui/components/text-input';

import Setting, {REQUIRED_MESSAGE, fieldLabel} from 'components/setting';

type InputType = 'input' | 'number' | 'textarea' | 'date' | 'datetime-local';

type Props = {
    id?: string;
    label?: React.ReactNode;
    placeholder?: string;
    value?: string | number | null;
    addValidate?: (isValid: () => boolean) => void;
    removeValidate?: (isValid: () => boolean) => void;
    maxLength?: number;
    onChange?(id: string | undefined, value: string | number): void;
    disabled?: boolean;
    required?: boolean;
    readOnly?: boolean;
    type?: InputType;
};

type State = {
    invalid: boolean;
};

export default class Input extends React.PureComponent<Props, State> {
    state: State = {invalid: false};

    componentDidMount() {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props) {
        if (this.state.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        if (!this.props.onChange) {
            return;
        }

        if (this.props.type === 'number') {
            this.props.onChange(this.props.id, parseInt(e.target.value, 10));
        } else {
            this.props.onChange(this.props.id, e.target.value);
        }
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value) && this.props.value?.toString().length !== 0;
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const {id, type = 'input', required} = this.props;
        const label = this.props.label && fieldLabel(this.props.label, required);
        const message = required && this.state.invalid ? REQUIRED_MESSAGE : null;
        const common = {
            id,
            placeholder: this.props.placeholder,
            value: this.props.value || '',
            maxLength: this.props.maxLength,
            disabled: this.props.disabled,
            readOnly: this.props.readOnly,
            onChange: this.handleChange,
            message,
        };

        if (type === 'textarea') {
            return (
                <Setting>
                    <TextArea
                        {...common}
                        label={label}
                        rows={5}
                    />
                </Setting>
            );
        }

        if (type === 'date' || type === 'datetime-local') {
            return (
                <Setting>
                    {label && (
                        <label
                            className='jira-setting__label'
                            htmlFor={id}
                        >
                            {label}
                        </label>
                    )}
                    <TextInput
                        {...common}
                        type={type}
                    />
                </Setting>
            );
        }

        return (
            <Setting>
                <TextInput
                    {...common}
                    label={label}
                    type={type === 'number' ? 'number' : 'text'}
                />
            </Setting>
        );
    }
}
