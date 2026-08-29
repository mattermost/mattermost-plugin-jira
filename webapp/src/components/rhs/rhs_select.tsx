// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {
    useEffect,
    useId,
    useRef,
    useState,
} from 'react';

export type RHSSelectOption<T extends string> = {
    value: T;
    label: string;
};

export type Props<T extends string> = {
    value: T;
    options: Array<RHSSelectOption<T>>;
    ariaLabel: string;
    testId: string;
    onChange: (value: T) => void;
};

export default function RHSSelect<T extends string>(props: Props<T>): JSX.Element {
    const {value, options, ariaLabel, testId, onChange} = props;
    const rootRef = useRef<HTMLDivElement>(null);
    const [open, setOpen] = useState(false);
    const reactId = useId();
    const listboxId = reactId + '-listbox';

    const selected = options.find((option) => option.value === value) || options[0];
    const selectedIndex = Math.max(0, options.findIndex((option) => option.value === value));
    const [activeIndex, setActiveIndex] = useState(selectedIndex);

    useEffect(() => {
        setActiveIndex(selectedIndex);
    }, [selectedIndex, open]);

    useEffect(() => {
        const onDoc = (event: MouseEvent) => {
            if (!open) {
                return;
            }
            if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
                setOpen(false);
            }
        };
        document.addEventListener('mousedown', onDoc);
        return () => {
            document.removeEventListener('mousedown', onDoc);
        };
    }, [open]);

    const optionId = (index: number) => {
        return listboxId + '-opt-' + String(index);
    };

    const selectAt = (index: number) => {
        const option = options[index];
        if (!option) {
            return;
        }
        onChange(option.value);
        setOpen(false);
    };

    const onButtonKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
        if (event.key === 'Escape') {
            setOpen(false);
            return;
        }
        if (event.key === 'ArrowDown') {
            event.preventDefault();
            if (!open) {
                setOpen(true);
                return;
            }
            setActiveIndex((current) => Math.min(current + 1, options.length - 1));
            return;
        }
        if (event.key === 'ArrowUp') {
            event.preventDefault();
            if (!open) {
                setOpen(true);
                return;
            }
            setActiveIndex((current) => Math.max(current - 1, 0));
            return;
        }
        if (event.key === 'Home' && open) {
            event.preventDefault();
            setActiveIndex(0);
            return;
        }
        if (event.key === 'End' && open) {
            event.preventDefault();
            setActiveIndex(options.length - 1);
            return;
        }
        if ((event.key === 'Enter' || event.key === ' ') && open) {
            event.preventDefault();
            selectAt(activeIndex);
            return;
        }
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            setOpen(true);
        }
    };

    return (
        <div
            className='jira-rhs-select'
            ref={rootRef}
        >
            <button
                type='button'
                className='jira-rhs-select-button'
                aria-label={ariaLabel}
                aria-haspopup='listbox'
                aria-expanded={open}
                aria-controls={listboxId}
                {...(open ? {'aria-activedescendant': optionId(activeIndex)} : {})}
                data-testid={testId}
                onClick={() => setOpen((current) => !current)}
                onKeyDown={onButtonKeyDown}
            >
                <span className='jira-rhs-select-label'>{selected ? selected.label : ''}</span>
                <i
                    className='icon icon-chevron-down'
                    aria-hidden={true}
                />
            </button>
            {open && (
                <div
                    className='jira-rhs-select-menu'
                    id={listboxId}
                    role='listbox'
                    aria-label={ariaLabel}
                >
                    {options.map((option, index) => {
                        const isSelected = option.value === value;
                        return (
                            <div
                                key={option.value}
                                id={optionId(index)}
                                className={index === activeIndex ? 'jira-rhs-select-option jira-rhs-select-option--active' : 'jira-rhs-select-option'}
                                role='option'
                                aria-selected={isSelected}
                                onMouseEnter={() => setActiveIndex(index)}
                                onClick={() => selectAt(index)}
                            >
                                {option.label}
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
