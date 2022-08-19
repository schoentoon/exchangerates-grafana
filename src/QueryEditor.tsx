import { defaults } from 'lodash';

import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { defaultQuery, MyDataSourceOptions, MyQuery } from './types';

const { FormField } = LegacyForms;

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  onBaseCurrencyChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, baseCurrency: event.target.value });
  };

  onToCurrencyChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, toCurrency: event.target.value });
    // executes the query
    onRunQuery();
  };

  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { baseCurrency, toCurrency } = query;

    return (
      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={baseCurrency || ''}
          onChange={this.onBaseCurrencyChange}
          label="Base currency"
        />
        <FormField
          width={4}
          value={toCurrency || ''}
          onChange={this.onToCurrencyChange}
          label="To currency"
        />
      </div>
    );
  }
}
