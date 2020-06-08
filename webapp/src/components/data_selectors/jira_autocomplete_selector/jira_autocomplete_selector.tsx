// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import AsyncSelect from 'react-select/async';
import debounce from 'debounce-promise';

import Setting from 'components/setting';
import {getStyleForReactSelect} from 'utils/styles';
import {ReactSelectOption} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

const searchDebounceDelay = 400;

const stripHTML = (text: string) => {
    if (!text) {
        return text;
    }

    var doc = new DOMParser().parseFromString(text, 'text/html');
    return doc.body.textContent || '';
 }

type Props = BackendSelectorProps & {
    searchAutoCompleteFields: (params: object) => Promise<any>;
    fieldName: string;
};

export default class JiraAutoCompleteSelector extends React.PureComponent<Props> {
    fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!this.props.value || (this.props.isMulti && !this.props.value.length)) {
            return [];
        }

        return this.searchAutoCompleteFields('');
    };

    searchAutoCompleteFields = (inputValue: string): Promise<ReactSelectOption[]> => {
        const {fieldName} = this.props;
        const params = {
            fieldValue: inputValue,
            fieldName,
        };
        return this.props.searchAutoCompleteFields(params).then(({data}) => {
            return data.results.map((suggestion: {value: string, displayName: string}) => ({
                value: suggestion.value,
                label: stripHTML(suggestion.displayName),
            }));
        });
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...this.props}
                name={'epic'}
                fetchInitialSelectedValues={this.fetchInitialSelectedValues}
                search={this.searchAutoCompleteFields}
            />
        );
    }
}
