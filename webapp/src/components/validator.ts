// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

type ValidateFn = () => boolean;

export default class Validator {
    private components: ValidateFn[] = [];

    addComponent = (validateField: ValidateFn): void => {
        this.components.push(validateField);
    };

    removeComponent = (validateField: ValidateFn): void => {
        const index = this.components.indexOf(validateField);
        if (index !== -1) {
            this.components.splice(index, 1);
        }
    };

    validate = (): boolean => {
        return Array.from(this.components.values()).reduce((accum, validateField) => {
            return validateField() && accum;
        }, true);
    };
}
