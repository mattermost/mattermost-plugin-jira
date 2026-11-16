// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {CloseCircleIcon} from '@mattermost/compass-icons/components';
import {DateRangePicker} from '@mattermost/compass-ui/components/date-range-picker';
import {Icon} from '@mattermost/compass-ui/components/icon';
import {IconButton} from '@mattermost/compass-ui/components/icon-button';
import {TextArea} from '@mattermost/compass-ui/components/text-area';
import {TextInput} from '@mattermost/compass-ui/components/text-input';

import Setting, {REQUIRED_MESSAGE, fieldLabel} from 'components/setting';
import WithTooltip from 'components/with_tooltip';

import {Z_INDEX_MENU_PORTAL} from 'utils/styles';

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

function hasValue(value: Props['value']): value is string | number {
    if (value === null || typeof value === 'undefined' || Number.isNaN(value)) {
        return false;
    }
    return value.toString().length !== 0;
}

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

    changeValue = (value: string) => {
        this.props.onChange?.(this.props.id, value);
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = hasValue(this.props.value);
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
            value: hasValue(this.props.value) ? this.props.value : '',
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
            const [date = '', time = ''] = common.value.toString().split('T');
            const locked = this.props.disabled || this.props.readOnly;
            const labelId = id && `${id}-label`;
            let errorId;
            if (message && id) {
                errorId = `${id}-error`;
            }

            return (
                <Setting
                    error={message}
                    errorId={errorId}
                >
                    {label && (
                        <span
                            id={labelId}
                            className='jira-setting__label'
                        >
                            {label}
                        </span>
                    )}
                    <div
                        role='group'
                        aria-labelledby={labelId}
                        aria-describedby={errorId}
                        className='jira-setting__date'
                    >
                        <DateRangePicker
                            id={id}
                            value={date}
                            disabled={locked}
                            zIndex={Z_INDEX_MENU_PORTAL}
                            onChange={(nextDate) => this.changeValue(type === 'date' ? nextDate : `${nextDate}T${time || '00:00'}`)}
                        />
                        {type === 'datetime-local' && (
                            <TextInput
                                aria-label='Time'
                                type='time'
                                value={time}
                                disabled={locked || !date}
                                onChange={(e) => this.changeValue(`${date}T${e.target.value}`)}
                            />
                        )}
                        {date && !locked && (
                            <WithTooltip label='Clear'>
                                <IconButton
                                    aria-label={typeof this.props.label === 'string' ? `Clear ${this.props.label}` : 'Clear date'}
                                    size='small'
                                    icon={<Icon glyph={<CloseCircleIcon/>}/>}
                                    onClick={() => this.changeValue('')}
                                />
                            </WithTooltip>
                        )}
                    </div>
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
