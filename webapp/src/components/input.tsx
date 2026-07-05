// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent, ChangeEvent} from 'react';

import Setting from './setting';

type InputType = 'number' | 'input' | 'textarea' | 'date' | 'datetime-local';

type Props = {
    id?: string;
    label: React.ReactNode;
    placeholder?: string;
    helpText?: React.ReactNode;
    value?: string | number;
    addValidate: (fn: () => boolean) => void;
    removeValidate: (fn: () => boolean) => void;
    maxLength?: number | null;
    onChange?: (id: string, value: string | number) => void;
    disabled?: boolean;
    required?: boolean;
    readOnly?: boolean;
    type?: InputType;
};

type State = {
    invalid: boolean;
};

export default class Input extends PureComponent<Props, State> {
    static defaultProps = {
        type: 'input' as InputType,
        maxLength: null,
        required: false,
        readOnly: false,
    };

    constructor(props: Props) {
        super(props);
        this.state = {invalid: false};
    }

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

    componentDidUpdate(prevProps: Props, prevState: State) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        if (this.props.type === 'number') {
            const numValue = e.target.value === '' ? '' : Number(e.target.value);
            this.props.onChange?.(this.props.id ?? '', numValue);
        } else {
            this.props.onChange?.(this.props.id ?? '', e.target.value);
        }
    };

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }
        const {value} = this.props;
        const valid = value !== undefined && value !== null && value !== '';
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const requiredMsg = 'This field is required.';
        const style = getStyle();
        const value = this.props.value ?? '';

        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let input = null;
        if (this.props.type === 'input') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='text'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength ?? undefined}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'number') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='number'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength ?? undefined}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'textarea') {
            input = (
                <textarea
                    style={style.textarea}
                    resize='none'
                    id={this.props.id}
                    className='form-control'
                    rows={5}
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength ?? undefined}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'date' || this.props.type === 'datetime-local') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type={this.props.type}
                    placeholder={this.props.placeholder}
                    value={value}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        }

        return (
            <Setting
                label={this.props.label}
                helpText={this.props.helpText}
                inputId={this.props.id}
                required={this.props.required}
            >
                {input}
                {validationError}
            </Setting>
        );
    }
}

const getStyle = () => ({
    textarea: {
        resize: 'none' as const,
    },
});
